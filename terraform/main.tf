resource "rediscloud_database" "test" {
  subscription_id              = 1726714
  name                         = "tf-database-02"
  protocol                     = "redis"
  memory_limit_in_gb           = 0.1
  data_persistence             = "none"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 10000
  password                     = "changeMe"

  alert {
    name  = "dataset-size"
    value = 40
  }
}

provider "rediscloud" {
  url        = "https://api.redislabs.com/v1"
  api_key    = "A29yz82qyuvtam52r8opa1902h98xgj3i3r2vx6s76pfi6pq2gt"
  secret_key = "S2gs1xpo1k6qc7plc902evl5r78ksgfqvshcfw7uts93awyy3oy"
}
