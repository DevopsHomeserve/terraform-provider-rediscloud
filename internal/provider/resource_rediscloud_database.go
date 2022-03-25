package provider

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudDatabase() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates a Subscription and database resources within your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudDatabaseCreate,
		ReadContext:   resourceRedisCloudSDatabaseRead,
		UpdateContext: resourceRedisCloudDatabaseUpdate,
		DeleteContext: resourceRedisCloudDatabaseDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description:      "ID of the subscription that the database belongs to",
				Type:             schema.TypeString,
				ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				Required:         true,
			},
			"db_id": {
				Description: "Identifier of the database created",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"name": {
				Description:      "A meaningful name to identify the database",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 40)),
			},
			"protocol": {
				Description:      "The protocol that will be used to access the database, (either ‘redis’ or 'memcached’) ",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.ProtocolValues(), false)),
			},
			"memory_limit_in_gb": {
				Description: "Maximum memory usage for this specific database",
				Type:        schema.TypeFloat,
				Required:    true,
			},
			"support_oss_cluster_api": {
				Description: "Support Redis open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"external_endpoint_for_oss_cluster_api": {
				Description: "Should use the external endpoint for open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"data_persistence": {
				Description: "Rate of database data persistence (in persistent storage)",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "none",
			},
			"replication": {
				Description: "Databases replication",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"throughput_measurement_by": {
				Description:      "Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"number-of-shards", "operations-per-second"}, false)),
			},
			"throughput_measurement_value": {
				Description: "Throughput value (as applies to selected measurement method)",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"average_item_size_in_bytes": {
				Description: "Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes) of the items stored in the database",
				Type:        schema.TypeInt,
				Optional:    true,
				// Setting default to 0 so that the hash func produces the same hash when this field is not
				// specified. SDK's catch-all issue around this: https://github.com/hashicorp/terraform-plugin-sdk/issues/261
				Default: 0,
			},
			"password": {
				Description: "Password used to access the database",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"public_endpoint": {
				Description: "Public endpoint to access the database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"private_endpoint": {
				Description: "Private endpoint to access the database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"client_ssl_certificate": {
				Description: "SSL certificate to authenticate user connections",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"periodic_backup_path": {
				Description: "Path that will be used to store database backup files",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"replica_of": {
				Description: "Set of Redis database URIs, in the format `redis://user:password@host:port`, that this database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateDiagFunc(validation.IsURLWithScheme([]string{"redis"})),
				},
			},
			"alert": {
				Description: "Set of alerts to enable on the database",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description:      "Alert name",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.AlertNameValues(), false)),
						},
						"value": {
							Description: "Alert value",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
			"module": {
				Description: "A module object",
				Type:        schema.TypeList,
				Optional:    true,
				MinItems:    1,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of the module to enable",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"source_ips": {
				Description: "Set of CIDR addresses to allow access to the database",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
				},
			},
			"hashing_policy": {
				Description: "List of regular expression rules to shard the database by. See the documentation on clustering for more information on the hashing policy - https://docs.redislabs.com/latest/rc/concepts/clustering/",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					// Can't check that these are valid regex rules as the service wants something like `(?<tag>.*)`
					// which isn't a valid Go regex
				},
			},
			"enable_tls": {
				Description: "Use TLS for authentication",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
}
func resourceRedisCloudDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	var alerts []*databases.CreateAlert
	for _, alert := range d.Get("alert").(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		alerts = append(alerts, &databases.CreateAlert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	createModules := make([]*databases.CreateModule, 0)
	module := d.Get("module")
	for _, module := range module.([]interface{}) {
		moduleMap := module.(map[string]interface{})

		modName := moduleMap["name"].(string)

		createModule := &databases.CreateModule{
			Name: redis.String(modName),
		}

		createModules = append(createModules, createModule)
	}

	create := databases.CreateDatabase{
		DryRun:               redis.Bool(false),
		Name:                 redis.String(d.Get("name").(string)),
		Protocol:             redis.String(d.Get("protocol").(string)),
		MemoryLimitInGB:      redis.Float64(d.Get("memory_limit_in_gb").(float64)),
		SupportOSSClusterAPI: redis.Bool(d.Get("support_oss_cluster_api").(bool)),
		DataPersistence:      redis.String(d.Get("data_persistence").(string)),
		Replication:          redis.Bool(d.Get("replication").(bool)),
		ThroughputMeasurement: &databases.CreateThroughputMeasurement{
			By:    redis.String(d.Get("throughput_measurement_by").(string)),
			Value: redis.Int(d.Get("throughput_measurement_value").(int)),
		},
		Alerts:    alerts,
		ReplicaOf: setToStringSlice(d.Get("replica_of").(*schema.Set)),
		Password:  redis.String(d.Get("password").(string)),
		SourceIP:  setToStringSlice(d.Get("source_ips").(*schema.Set)),
		Modules:   createModules,
	}

	averageItemSize := d.Get("average_item_size_in_bytes").(int)
	if averageItemSize > 0 {
		create.AverageItemSizeInBytes = redis.Int(averageItemSize)
	}

	// The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := d.Get("client_ssl_certificate").(string)
	enableTLS := d.Get("enable_tls").(bool)
	if enableTLS {
		// TLS only: enable_tls=true, client_ssl_certificate="".
		create.EnableTls = redis.Bool(enableTLS)
		// mTLS: enableTls=true, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			create.ClientSSLCertificate = redis.String(clientSSLCertificate)
		}
	} else {
		// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			create.ClientSSLCertificate = redis.String(clientSSLCertificate)
		} else {
			// Default: enable_tls=false, client_ssl_certificate=""
			create.EnableTls = redis.Bool(enableTLS)
		}
	}

	backupPath := d.Get("periodic_backup_path").(string)
	if backupPath != "" {
		create.PeriodicBackupPath = redis.String(backupPath)
	}

	// if v, ok := d.Get("external_endpoint_for_oss_cluster_api"); ok {
	// 	create.UseExternalEndpointForOSSClusterAPI = redis.Bool(v.(bool))
	// }
	id, err := api.client.Database.Create(ctx, subId, create)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(id))

	log.Printf("[DEBUG] Created database %d", id)

	if err := waitForDatabaseToBeActive(ctx, subId, id, api); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudSDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(db *databases.Database) bool
	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.StringValue(db.Name) == v.(string)
		})
	}
	if v, ok := d.GetOk("protocol"); ok {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.StringValue(db.Protocol) == v.(string)
		})
	}

	list := api.client.Database.List(ctx, subId)
	dbs, err := filterDatabases(list, filters)
	if err != nil {
		return diag.FromErr(list.Err())
	}

	if len(dbs) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(dbs) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	// Some attributes are only returned when retrieving a single database
	db, err := api.client.Database.Get(ctx, subId, redis.IntValue(dbs[0].ID))
	if err != nil {
		return diag.FromErr(list.Err())
	}

	d.SetId(fmt.Sprintf("%d", redis.IntValue(db.ID)))

	if err := d.Set("name", redis.StringValue(db.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("protocol", redis.StringValue(db.Protocol)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory_limit_in_gb", redis.Float64Value(db.MemoryLimitInGB)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("support_oss_cluster_api", redis.BoolValue(db.SupportOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("data_persistence", redis.StringValue(db.DataPersistence)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("replication", redis.BoolValue(db.Replication)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("throughput_measurement_by", redis.StringValue(db.ThroughputMeasurement.By)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("throughput_measurement_value", redis.IntValue(db.ThroughputMeasurement.Value)); err != nil {
		return diag.FromErr(err)
	}
	if v := redis.StringValue(db.Security.Password); v != "" {
		if err := d.Set("password", v); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("public_endpoint", redis.StringValue(db.PublicEndpoint)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("private_endpoint", redis.StringValue(db.PrivateEndpoint)); err != nil {
		return diag.FromErr(err)
	}
	if db.ReplicaOf != nil {
		if err := d.Set("replica_of", redis.StringSliceValue(db.ReplicaOf.Endpoints...)); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("alert", flattenAlerts(db.Alerts)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("module", flattenModules(db.Modules)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("hashing_policy", flattenRegexRules(db.Clustering.RegexRules)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	databaseId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting database %d on subscription %d", databaseId, subId)

	dbErr := api.client.Database.Delete(ctx, subId, databaseId)
	if dbErr != nil {
		diag.FromErr(dbErr)
	}

	d.SetId("")

	return diags
}

func resourceRedisCloudDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)
	var alerts []*databases.UpdateAlert
	for _, alert := range d.Get("alert").(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		alerts = append(alerts, &databases.UpdateAlert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	update := databases.UpdateDatabase{
		Name:                 redis.String(d.Get("name").(string)),
		MemoryLimitInGB:      redis.Float64(d.Get("memory_limit_in_gb").(float64)),
		SupportOSSClusterAPI: redis.Bool(d.Get("support_oss_cluster_api").(bool)),
		Replication:          redis.Bool(d.Get("replication").(bool)),
		ThroughputMeasurement: &databases.UpdateThroughputMeasurement{
			By:    redis.String(d.Get("throughput_measurement_by").(string)),
			Value: redis.Int(d.Get("throughput_measurement_value").(int)),
		},
		DataPersistence: redis.String(d.Get("data_persistence").(string)),
		Password:        redis.String(d.Get("password").(string)),
		SourceIP:        setToStringSlice(d.Get("source_ips").(*schema.Set)),
		Alerts:          alerts,
	}

	update.ReplicaOf = setToStringSlice(d.Get("replica_of").(*schema.Set))
	if update.ReplicaOf == nil {
		update.ReplicaOf = make([]*string, 0)
	}

	// The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := d.Get("client_ssl_certificate").(string)
	enableTLS := d.Get("enable_tls").(bool)
	if enableTLS {
		// TLS only: enable_tls=true, client_ssl_certificate="".
		update.EnableTls = redis.Bool(enableTLS)
		// mTLS: enableTls=true, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
		}
	} else {
		// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
		} else {
			// Default: enable_tls=false, client_ssl_certificate=""
			update.EnableTls = redis.Bool(enableTLS)
		}
	}

	regex := d.Get("hashing_policy").([]interface{})
	if len(regex) != 0 {
		update.RegexRules = interfaceToStringSlice(regex)
	}

	backupPath := d.Get("periodic_backup_path").(string)
	if backupPath != "" {
		update.PeriodicBackupPath = redis.String(backupPath)
	}

	// if v, ok := d.Get("external_endpoint_for_oss_cluster_api"); ok {
	// 	update.UseExternalEndpointForOSSClusterAPI = redis.Bool(v.(bool))
	// }
	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	databaseId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Updating database %s (%d)", redis.StringValue(update.Name), databaseId)

	err = api.client.Database.Update(ctx, subId, databaseId, update)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}
