# Creating the job runs the mandatory dry-run only. Review the dry-run (status,
# graphiant_playbook_job_logs), then set approve = true in a follow-up change to
# deploy it.
resource "graphiant_playbook_job" "ntp" {
  config_id = graphiant_playbook_config.ntp.id
  approve   = false

  # Start a fresh dry-run whenever the config's files change.
  triggers = {
    files = sha256(jsonencode(graphiant_playbook_config.ntp.files))
  }
}
