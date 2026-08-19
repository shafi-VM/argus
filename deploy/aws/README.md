# Deploy Argus to AWS (Terraform, one box)

A lightweight Terraform module that stands the Argus stack up on a single EC2 host, and
tears it down again. It exists so **deployability is a permanent, reproducible artifact** —
not so something runs 24/7. Treat AWS credit as *on-demand demo fuel*: `apply` for a demo
window, show it live, `destroy` when you're done.

> **Status — honest scope.** This module is written and validated offline
> (`terraform fmt` + `terraform validate` pass). It has **not** been `apply`-ed against a
> live AWS account — your first `terraform apply` is the end-to-end test. Cloud-init reliably
> brings up **Tier-1 (PREVENT + Mission Control)**; the SigNoz-backed **LEARN** tier is a
> documented follow-on (below), because its API key can only be minted after SigNoz's first
> boot.

## What it creates

- One EC2 instance (Ubuntu 24.04 LTS, `t3.large` by default), gp3 root volume.
- A security group scoped to **your IP** — SSH (22), SigNoz UI (8081), Argus (8088).
- Cloud-init that installs Docker, clones this repo, and runs `docker compose up -d --build`
  → the PREVENT demo is live at `http://<ip>:8088/mission` a couple minutes after boot.

## Prerequisites

- An AWS account with the [CLI configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html) (`aws sts get-caller-identity` works).
- [Terraform](https://developer.hashicorp.com/terraform/install) ≥ 1.5.
- An EC2 key pair for SSH:
  ```bash
  aws ec2 create-key-pair --key-name argus \
    --query KeyMaterial --output text > argus.pem && chmod 400 argus.pem
  ```

## Bring it up

```bash
cd deploy/aws
cp terraform.tfvars.example terraform.tfvars      # set allowed_cidr + key_name
#   allowed_cidr = "$(curl -s ifconfig.me)/32"     <- your IP only, never 0.0.0.0/0

terraform init
terraform plan          # review
terraform apply         # ~1 min to provision; cloud-init then runs ~2-3 min

terraform output        # public IP + URLs + a tail-the-bootstrap command
```

Open the `mission_control_url`, or SSH in and run the money moment:

```bash
ssh -i argus.pem ubuntu@<public_ip>
cd /opt/argus && make demo
```

## Enable the LEARN tier (SigNoz)

Tier-1 needs no backend. For the full **drift → quarantine → reroute → recover** arc, stand
up SigNoz on the box and point Argus at it — the same steps as [`../../DEPLOY.md`](../../DEPLOY.md):

```bash
ssh -i argus.pem ubuntu@<public_ip>
cd /opt/argus
# 1. Bring up SigNoz (Foundry) — creates the 'signoz-network' the Argus overlay joins.
#    Follow ../../DEPLOY.md §1. First boot runs migrations (~2-3 min).
# 2. Open the SigNoz UI (http://<ip>:8081), create an account, mint an API key.
# 3. echo "SIGNOZ_API_KEY=<key>" > .env
# 4. make demo-signoz        # PREVENT + LEARN, provisions the hero dashboard
```

## Cost & teardown

`t3.large` + 40 GB gp3 is roughly **$60–65/month if left running** — so a $100 credit lasts
weeks, not forever, and AWS credits often expire. **Don't leave it up.** Per-hour on-demand
use (apply → demo → destroy) is a few cents, so the credit covers many demos.

```bash
terraform destroy       # removes the instance, volume, and security group
```

## Security notes

- `allowed_cidr` **must** be your own IP (`/32`). The module refuses `0.0.0.0/0`: the Argus
  proxy forwards the caller's upstream credentials, and an open SigNoz is world-readable.
- The default subnet gives the box a public IP. For anything beyond a demo, front it with a
  reverse proxy + TLS and real auth — out of scope for this on-demand module.
