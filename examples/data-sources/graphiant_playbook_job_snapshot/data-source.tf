# The module files the job actually ran with, even if the config was edited since.
data "graphiant_playbook_job_snapshot" "ntp" {
  job_id = graphiant_playbook_job.ntp.id
}
