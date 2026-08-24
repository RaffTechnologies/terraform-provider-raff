package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceAppService() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Raff Apps service — a web service, private service, worker, " +
			"cron job, or one-off job on the Raff PaaS. Code deploys happen outside " +
			"Terraform — push with the `raff apps deploy` CLI, the dashboard, or a " +
			"connected GitHub repo. Scaling changes on a live service roll out with no " +
			"downtime.",
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
				ForceNew:    true,
				Description: "Display name, unique within the project.",
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
				ForceNew: true,
			},
			"source_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"git", "image", "template", "compose"}, false),
				Description:  "Where the service's code comes from: `git`, `image`, `template`, or `compose`.",
			},
			"builder": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"buildpacks", "dockerfile", "image"}, false),
				Description:  "Build strategy: `buildpacks`, `dockerfile`, or `image`.",
			},
			"image_ref": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Prebuilt image reference (for `source_type = image`).",
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
				ForceNew:    true,
				Description: "Git branch to deploy.",
			},
			"repo_root_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Source root directory within the repository.",
			},
			"dockerfile_path": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Path to the Dockerfile (for `builder = dockerfile`).",
			},
			"start_command": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Override the container start command.",
			},
			"tier_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Pricing tier ID (see the `raff_app_tiers` data source).",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Deployment region.",
			},
			"cron_schedule": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Cron schedule (for `service_type = cron`).",
			},
			"cron_timezone": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Cron timezone (for `service_type = cron`).",
			},
			"http_port": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Container port the service listens on.",
			},
			"health_check_path": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
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
				Description: "URL label — the name plus a short random suffix.",
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
				Description: "In-cluster hostname (private services).",
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
	return setAppServiceState(d, s)
}

func resourceAppServiceUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	if !d.HasChanges("replicas", "autoscaling_enabled", "min_replicas", "max_replicas", "scale_to_zero") {
		return resourceAppServiceRead(ctx, d, meta)
	}

	req := &raff.ScaleAppServiceRequest{
		Replicas:    d.Get("replicas").(int),
		ScaleToZero: d.Get("scale_to_zero").(bool),
	}
	if d.HasChanges("autoscaling_enabled", "min_replicas", "max_replicas") {
		req.AutoscalingSet = true
		req.Autoscaling = d.Get("autoscaling_enabled").(bool)
		req.MinReplicas = d.Get("min_replicas").(int)
		req.MaxReplicas = d.Get("max_replicas").(int)
	}

	s, _, err := client.AppServices.Scale(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}
	return setAppServiceState(d, s)
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
