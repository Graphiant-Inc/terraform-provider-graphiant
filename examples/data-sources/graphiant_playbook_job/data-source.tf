# Latest job for a config; use job_id instead to look up a specific job.
data "graphiant_playbook_job" "latest" {
  config_id = graphiant_playbook_config.ntp.id
}
