output "public_ip" {
  description = "Public IP of the Argus box."
  value       = aws_instance.argus.public_ip
}

output "ssh" {
  description = "SSH into the box."
  value       = "ssh ubuntu@${aws_instance.argus.public_ip}"
}

output "mission_control_url" {
  description = "Argus Mission Control — the PREVENT money moment (up ~2-3 min after apply)."
  value       = "http://${aws_instance.argus.public_ip}:8088/mission"
}

output "signoz_url" {
  description = "SigNoz UI — only after you enable the LEARN tier (see deploy/aws/README.md)."
  value       = "http://${aws_instance.argus.public_ip}:8081"
}

output "tail_bootstrap" {
  description = "Watch cloud-init bring the stack up."
  value       = "ssh ubuntu@${aws_instance.argus.public_ip} 'tail -f /var/log/argus-bootstrap.log'"
}
