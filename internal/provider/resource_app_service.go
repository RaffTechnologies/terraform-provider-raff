package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceAppService() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Raff Apps app: a web service, private service, worker, cron job or one-off job. Image and GitHub apps deploy when created. Settings change in place, a new `image_ref` rolls out as a deploy, and scaling changes apply to the running app. An app built from uploaded source deploys when you upload it with `raff apps deploy --dir`.",
		CreateContext: resourceAppServiceCreate,
		ReadContext:   resourceAppServiceRead,
		UpdateContext: resourceAppServiceUpdate,
		DeleteContext: resourceAppServiceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name, unique within the project. Renames in place; the app's address does not change.",
			},
			"service_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"web", "private", "worker", "cron", "job"}, false),
				Description:  "Service type: `web`, `private`, `worker`, `cron`, or `job`.",
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"source_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"image", "git", "console", "template", "compose"}, false),
				Description:  "Where the code comes from: `image` (a Docker image or a template's image), `git` (a GitHub repository) or `console` (source uploaded with `raff apps deploy --dir`). Image and GitHub apps deploy when created.",
			},
			"builder": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"buildpacks", "dockerfile", "image"}, false),
				Description:  "Build strategy: `buildpacks`, `dockerfile`, or `image`.",
			},
			"image_ref": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Prebuilt image reference (for `source_type = image`). A change rolls out as a deploy of the new image.",
			},
			"repo_full_name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "GitHub repository (`owner/name`) for `source_type = git`.",
			},
			"repo_branch": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Git branch to deploy.",
			},
			"repo_root_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Source root directory within the repository.",
			},
			"dockerfile_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Path to the Dockerfile (for `builder = dockerfile`).",
			},
			"start_command": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Override the container start command.",
			},
			"tier_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Pricing tier ID (see the `raff_app_tiers` data source). Changes in place; the app restarts on the new size.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Deployment region.",
			},
			"cron_schedule": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Cron schedule (for `service_type = cron`).",
			},
			"cron_timezone": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Cron timezone (for `service_type = cron`).",
			},
			"http_port": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Container port the service listens on.",
			},
			"health_check_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "HTTP path used for health checks.",
			},
			// Updatable via scale
			"replicas": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Fixed replica count. Ignored when autoscaling is enabled.",
			},
			"autoscaling_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable autoscaling between `min_replicas` and `max_replicas`.",
			},
			"min_replicas": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Minimum replicas when autoscaling.",
			},
			"max_replicas": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Maximum replicas when autoscaling.",
			},
			"scale_to_zero": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Scale to zero when idle (web/private services).",
			},
			// Computed
			"cron_concurrency": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"forbid", "allow", "replace"}, false),
				Description:  "Cron services: what happens when a run is due while the last one still runs: `forbid` (default), `allow` or `replace`.",
			},
			"github_installation_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "For `source_type = git`: the ID of a GitHub connection (`raff apps github`).",
			},
			"link_database_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "ID of a managed Postgres (`raff_database.<name>.id`) to link before the first deploy. The app waits until the database runs, then deploys with `DATABASE_URL` set. To link a database later, use `raff_app_database_link` instead.",
			},
			"env": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Environment variables. A change redeploys the app. Only the keys you set are tracked: variables a database link sets, and ones added outside Terraform, are left alone.",
			},
			"secret_env": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Environment variables stored as secrets (hidden on the app page). A change redeploys the app.",
			},
			"service_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Short reference id.",
			},
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Project this service belongs to.",
			},
			"slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL label made from the name.",
			},
			"tier_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Public HTTPS endpoint (web services; automatic TLS).",
			},
			"internal_host": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Address your other apps use to reach a private service, on its port.",
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status_message": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"billing_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAppServiceCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	s, _, err := client.AppServices.Create(ctx, &raff.CreateAppServiceRequest{
		Name:            d.Get("name").(string),
		Description:     d.Get("description").(string),
		ServiceType:     d.Get("service_type").(string),
		SourceType:      d.Get("source_type").(string),
		Builder:         d.Get("builder").(string),
		ImageRef:        d.Get("image_ref").(string),
		RepoFullName:    d.Get("repo_full_name").(string),
		RepoBranch:      d.Get("repo_branch").(string),
		RepoRootDir:     d.Get("repo_root_dir").(string),
		DockerfilePath:  d.Get("dockerfile_path").(string),
		StartCommand:    d.Get("start_command").(string),
		TierID:          d.Get("tier_id").(int),
		Replicas:        d.Get("replicas").(int),
		Autoscaling:     d.Get("autoscaling_enabled").(bool),
		MinReplicas:     d.Get("min_replicas").(int),
		MaxReplicas:     d.Get("max_replicas").(int),
		ScaleToZero:     d.Get("scale_to_zero").(bool),
		HTTPPort:        d.Get("http_port").(int),
		HealthCheckPath: d.Get("health_check_path").(string),
		CronSchedule:    d.Get("cron_schedule").(string),
		CronTimezone:    d.Get("cron_timezone").(string),
		Region:          d.Get("region").(string),

		CronConcurrency:      d.Get("cron_concurrency").(string),
		GitHubInstallationID: d.Get("github_installation_id").(string),
		LinkDatabaseID:       d.Get("link_database_id").(string),
		EnvVars:              appEnvInputs(d),
		DeployNow:            appDeploysOnCreate(d.Get("source_type").(string), d.Get("builder").(string)),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(s.ID)
	return setAppServiceState(d, s)
}

func resourceAppServiceRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	s, _, err := client.AppServices.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if diags := setAppServiceState(d, s); diags.HasError() {
		return diags
	}
	// Only the variables Terraform manages are tracked: the ones a database
	// link sets, and ones added outside Terraform, are left alone.
	vars, _, err := client.AppServices.ListEnvVars(ctx, d.Id(), true)
	if err != nil {
		return diag.FromErr(err)
	}
	env, secret := managedAppEnv(vars, d.Get("env").(map[string]any), d.Get("secret_env").(map[string]any))
	d.Set("env", env)
	d.Set("secret_env", secret)
	return nil
}

func resourceAppServiceUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	settings := &raff.UpdateAppServiceRequest{}
	for key, dst := range map[string]**string{
		"name": &settings.Name, "description": &settings.Description, "start_command": &settings.StartCommand,
		"health_check_path": &settings.HealthCheckPath, "cron_schedule": &settings.CronSchedule,
		"cron_timezone": &settings.CronTimezone, "cron_concurrency": &settings.CronConcurrency,
		"repo_branch": &settings.RepoBranch, "repo_root_dir": &settings.RepoRootDir, "dockerfile_path": &settings.DockerfilePath,
	} {
		if d.HasChange(key) {
			*dst = raff.String(d.Get(key).(string))
		}
	}
	if d.HasChange("tier_id") {
		settings.TierID = raff.Int(d.Get("tier_id").(int))
	}
	if d.HasChange("http_port") {
		settings.HTTPPort = raff.Int(d.Get("http_port").(int))
	}
	if *settings != (raff.UpdateAppServiceRequest{}) {
		if _, _, _, err := client.AppServices.Update(ctx, d.Id(), settings); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("image_ref") {
		if _, _, err := client.AppServices.CreateDeployment(ctx, d.Id(), &raff.CreateAppDeploymentRequest{
			ImageRef: d.Get("image_ref").(string),
		}); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChanges("env", "secret_env") {
		if err := updateAppEnv(ctx, client, d); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChanges("replicas", "autoscaling_enabled", "min_replicas", "max_replicas", "scale_to_zero") {
		// With autoscaling_set the API takes autoscaling and scale to zero
		// exactly as sent, so the whole desired shape is sent every time.
		req := &raff.ScaleAppServiceRequest{
			Replicas:       d.Get("replicas").(int),
			AutoscalingSet: true,
			Autoscaling:    d.Get("autoscaling_enabled").(bool),
			MinReplicas:    d.Get("min_replicas").(int),
			MaxReplicas:    d.Get("max_replicas").(int),
			ScaleToZero:    d.Get("scale_to_zero").(bool),
		}
		if _, _, err := client.AppServices.Scale(ctx, d.Id(), req); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceAppServiceRead(ctx, d, meta)
}

func resourceAppServiceDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.AppServices.Delete(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setAppServiceState(d *schema.ResourceData, s *raff.AppService) diag.Diagnostics {
	d.Set("name", s.Name)
	d.Set("description", s.Description)
	d.Set("service_type", s.ServiceType)
	d.Set("source_type", s.SourceType)
	d.Set("builder", s.Builder)
	d.Set("image_ref", s.ImageRef)
	d.Set("repo_full_name", s.RepoFullName)
	d.Set("repo_branch", s.RepoBranch)
	d.Set("repo_root_dir", s.RepoRootDir)
	d.Set("dockerfile_path", s.DockerfilePath)
	d.Set("start_command", s.StartCommand)
	d.Set("tier_id", s.TierID)
	d.Set("tier_name", s.TierName)
	d.Set("region", s.Region)
	d.Set("cron_schedule", s.CronSchedule)
	d.Set("cron_timezone", s.CronTimezone)
	d.Set("cron_concurrency", s.CronConcurrency)
	d.Set("http_port", s.HTTPPort)
	d.Set("health_check_path", s.HealthCheckPath)
	d.Set("replicas", s.Replicas)
	d.Set("autoscaling_enabled", s.AutoscalingEnabled)
	d.Set("min_replicas", s.MinReplicas)
	d.Set("max_replicas", s.MaxReplicas)
	d.Set("scale_to_zero", s.ScaleToZero)
	d.Set("service_id", s.ServiceID)
	d.Set("project_id", s.ProjectID)
	d.Set("slug", s.Slug)
	d.Set("url", s.URL)
	d.Set("internal_host", s.InternalHost)
	d.Set("status", s.Status)
	d.Set("status_message", s.StatusMessage)
	d.Set("billing_type", s.BillingType)
	d.Set("created_at", s.CreatedAt)
	d.Set("updated_at", s.UpdatedAt)
	return nil
}

// appDeploysOnCreate: image and GitHub apps deploy as soon as they are
// created; an app built from uploaded source deploys when its source arrives.
func appDeploysOnCreate(sourceType, builder string) bool {
	return sourceType == "image" || sourceType == "git" || builder == "image"
}

func appEnvInputs(d *schema.ResourceData) []raff.AppEnvVarInput {
	var out []raff.AppEnvVarInput
	for k, v := range d.Get("env").(map[string]any) {
		out = append(out, raff.AppEnvVarInput{Key: k, Value: v.(string)})
	}
	for k, v := range d.Get("secret_env").(map[string]any) {
		out = append(out, raff.AppEnvVarInput{Key: k, Value: v.(string), IsSecret: true})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// managedAppEnv splits the app's variables into the env and secret_env maps,
// keeping only the keys Terraform manages (present in either map). Variables
// set by a database link are system variables and never managed.
func managedAppEnv(vars []raff.AppEnvVar, env, secret map[string]any) (map[string]string, map[string]string) {
	outEnv, outSecret := map[string]string{}, map[string]string{}
	for _, v := range vars {
		if v.IsSystem {
			continue
		}
		if _, ok := secret[v.Key]; ok {
			outSecret[v.Key] = v.Value
		} else if _, ok := env[v.Key]; ok {
			outEnv[v.Key] = v.Value
		}
	}
	return outEnv, outSecret
}

// updateAppEnv sets the new and changed variables in one call (one redeploy)
// and deletes the ones removed from the configuration.
func updateAppEnv(ctx context.Context, client *raff.Client, d *schema.ResourceData) error {
	var set []raff.SetAppEnvVarRequest
	want := map[string]bool{}
	for _, field := range []string{"env", "secret_env"} {
		oldRaw, newRaw := d.GetChange(field)
		oldMap, newMap := oldRaw.(map[string]any), newRaw.(map[string]any)
		for k, v := range newMap {
			want[k] = true
			if prev, ok := oldMap[k]; !ok || prev != v {
				set = append(set, raff.SetAppEnvVarRequest{Key: k, Value: v.(string), IsSecret: field == "secret_env"})
			}
		}
	}
	if len(set) > 0 {
		sort.Slice(set, func(i, j int) bool { return set[i].Key < set[j].Key })
		if _, _, err := client.AppServices.BulkSetEnvVars(ctx, d.Id(), &raff.BulkSetAppEnvVarsRequest{Vars: set}); err != nil {
			return err
		}
	}
	for _, field := range []string{"env", "secret_env"} {
		oldRaw, _ := d.GetChange(field)
		for k := range oldRaw.(map[string]any) {
			if !want[k] {
				if _, _, err := client.AppServices.DeleteEnvVar(ctx, d.Id(), k); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
