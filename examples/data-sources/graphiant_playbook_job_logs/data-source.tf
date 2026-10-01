data "graphiant_playbook_job_logs" "dry_run" {
  job_id = graphiant_playbook_job.ntp.id
  phase  = "dry_run"
}
