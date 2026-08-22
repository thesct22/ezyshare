# Deployment Steps

This document serves as a reference on how to deploy the backend as a Google Cloud Run job and the frontend on GitHub Pages.

## Backend Deployment on Google Cloud Run

### Setup GCP Foundation

1. Go to the [Google Cloud Console](https://console.cloud.google.com/) and create/select a project.
2. Enable Billing for your project, if not already enabled. You will need to create a billing account if you don't have one. 
3. Enable APIs for terraform scripts to use. To do this, you need to open the cloud shell (>_ icon on the top right corner of cloud console). This will start a new bash session on the browser. On the terminal run the following command:
    ```bash
    gcloud services enable run.googleapis.com artifactregistry.googleapis.com iam.googleapis.com cloudresourcemanager.googleapis.com
    ```

### Create Service Accounts

Terraform and GitHub actions will use this service account to deploy and manage resources on GCP.

1. In the search bar of the GCP Console, search for "Service Accounts" (under the "IAM & Admin" section)
2. Click on "Create Service Account"
3. Fill in the service account name (e.g. "github-actions-bot") and a description (e.g. "Service account for GitHub actions") and click "Create and Continue".
4. Grant it the following roles so it has the power to deploy and manage the resources we define in terraform:
    - roles/artifactregistry.writer
    - roles/run.admin
    - roles/iam.serviceAccountUser
    - roles/storage.objectAdmin
5. Click Continue and leave the principals with access section as is, and click Done.
6. Create a second service account for the Golang backend applications to write logs to cloud logging and access secrets from secret manager.
    - Service account name: "sa-ezyshare-backend"
7. Assign it the following roles so it can write logs to cloud logging and access secrets from secret manager:
    - roles/logging.logWriter
    - roles/secretmanager.secretAccessor
8. We will use workload identity federation instead of downloading the service account JSON file and using it as a secret in github actions. This is more secure and easier to manage.

### Setup gcloud and terraform CLI

To install gcloud CLI follow the instructions here: https://cloud.google.com/sdk/docs/install

To install terraform CLI follow the instructions here: https://developer.hashicorp.com/terraform/tutorials/gcp-get-started/install-cli

Once installed, run the following commands to authenticate gcloud:

```bash
# You can skip auth login and set project commands if you ran gcloud init command when following gcloud install instructions.
gcloud auth login
gcloud config set project <your-project-id>
gcloud config set compute/zone us-central1
gcloud config set compute/region us-central1
```

### Create Bucket for Terraform State

We will store terraform state in a bucket to allow us to manage infrastructure using terraform.

```bash
gcloud storage buckets create gs://<your-project-id>-tfstate \
--location=us-central1 \
--uniform-bucket-level-access

# Enable versioning so that we can recover if state gets lost or is corrupted
gcloud storage buckets update gs://<your-project-id>-tfstate \
--versioning
```

### Authenticate your Local Machine

Run the following command to authenticate gcloud on your local machine:

```bash
gcloud auth application-default login
```
This saves an ADC (Application Default Credentials) file locally which is used by Google's SDKs and Libraries (including Terraform and the Google Cloud Console) to authenticate requests made on your behalf. The ADC file is stored at `~/.config/gcloud/application_default_credentials.json`.

### Configure Workload Identity Federation for GitHub Actions

```bash
PROJECT_ID="<replace-with-your-project-id>"
GITHUB_REPO="<username/repository>"

# 1. Create a Workload Identity Pool
gcloud iam workload-identity-pools create github-actions-pool \
    --project="${PROJECT_ID}" \
    --location="global" \
    --display-name="GitHub Actions Pool"

# 2. Create a Workload Identity Provider for GitHub
gcloud iam workload-identity-pools providers create-oidc github-provider \
    --project="${PROJECT_ID}" \
    --location="global" \
    --workload-identity-pool="github-actions-pool" \
    --display-name="GitHub Actions Provider" \
    --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
    --attribute-condition="attribute.repository == '${GITHUB_REPO}'" \
    --issuer-uri="https://token.actions.githubusercontent.com"

# 3. Get the Pool ID (we need it for binding)
WORKLOAD_IDENTITY_POOL_ID=$(gcloud iam workload-identity-pools describe "github-actions-pool" \
    --project="${PROJECT_ID}" \
    --location="global" \
    --format="value(name)")

# 4. Allow your sepcific GitHub repo to impersonate github-actions-bot SA
gcloud iam service-accounts add-iam-policy-binding \
    "github-actions-bot@${PROJECT_ID}.iam.gserviceaccount.com" \
    --project="${PROJECT_ID}" \
    --role="roles/iam.workloadIdentityUser" \
    --member="principalSet://iam.googleapis.com/${WORKLOAD_IDENTITY_POOL_ID}/attribute.repository/${GITHUB_REPO}"
```

### Initial Terraform Provisioning Rationale (Breaking the Deadlock)

To prevent a "chicken-and-egg" deadlock where Cloud Run tries to pull a non-existent image before the Artifact Registry repository is created, `main.tf` initializes Cloud Run with Google's public starter image (`gcr.io/cloudrun/hello`). 

Because `main.tf` includes `lifecycle { ignore_changes = [ template[0].containers[0].image ] }`, Terraform provisions the infrastructure safely on day 1, and will never overwrite your live image deployments when updating code!

### Deploying & Updating the Backend

Whenever you make updates to your Go backend code, deploy the new revision using these 3 steps:

1. **Rebuild local Docker image:**
   ```bash
   docker compose build backend
   ```

2. **Tag and push to GCP Artifact Registry:**
   ```bash
   docker tag ghcr.io/thesct22/ezyshare-backend:latest us-central1-docker.pkg.dev/<your-project-id>/ezyshare-repo/backend:latest
   docker push us-central1-docker.pkg.dev/<your-project-id>/ezyshare-repo/backend:latest
   ```

3. **Deploy new revision to Cloud Run:**
   ```bash
   gcloud run deploy ezyshare-backend \
       --image=us-central1-docker.pkg.dev/<your-project-id>/ezyshare-repo/backend:latest \
       --region=us-central1
   ```

> **Note on Endpoints:** GCP Cloud Run's edge load balancer reserves `/healthz` internally. To verify your live backend deployment status, test the API or metrics endpoints:
> - ICE Servers API: `https://<your-service-url>.a.run.app/api/v1/ice-servers`
> - Prometheus Metrics: `https://<your-service-url>.a.run.app/metrics`
> - WebSocket Endpoint: `wss://<your-service-url>.a.run.app/ws`

### Terraform Deployment Instructions

1. Create a `deployment/terraform/terraform.tfvars` file locally (Git-ignored):
   ```hcl
   project_id = "<your-project-id>"
   ```

2. Initialize Terraform with your remote state GCS bucket name:
    This is the same bucket that we created above (Step 3 of GCP Foundation)
   ```bash
   cd deployment/terraform
   terraform init -backend-config="bucket=<your-project-id>-tfstate"
   ```

3. Format & Validate:
   ```bash
   terraform fmt
   terraform validate
   ```

4. Preview changes (Dry Run):
   ```bash
   terraform plan
   ```

5. Apply infrastructure to GCP:
   ```bash
   terraform apply
   ```

---

## Continuous Integration & Continuous Deployment (CI/CD)

The repository includes three automated GitHub Actions workflows in `.github/workflows/` pinned to immutable commit SHAs:

### Workflows Overview

1. **`deploy-frontend.yml` (Frontend Deployment to GitHub Pages)**
   - **Trigger:** Automated on `push` to `main` when code under `frontend/**` changes.
   - **Actions:** Builds the Vite React SPA, injects the backend WebSocket (`wss://${{ secrets.BACKEND_CNAME }}/ws`) and API URLs, and deploys the static build + `CNAME` directly to the `gh-pages` branch.

2. **`deploy-backend.yml` (Backend Continuous Deployment to Cloud Run)**
   - **Trigger:** Automated on `push` to `main` when code under `backend/**` changes, or manually via `workflow_dispatch`.
   - **Actions:** Authenticates keylessly to GCP via Workload Identity Federation (WIF), builds the multi-stage Go container image, tags it with `${{ github.sha }}` and `latest`, pushes to GCP Artifact Registry, and updates the live Cloud Run service revision.

3. **`terraform-infra.yml` (Infrastructure Management)**
   - **Trigger:** Manual trigger via `workflow_dispatch` (allowing admins to choose `plan` or `apply` from the GitHub Actions UI) or reusable via `workflow_call`.
   - **Actions:** Authenticates keylessly to GCP via WIF, initializes Terraform against your remote GCS state bucket (`${{ secrets.GCP_TFSTATE_BUCKET }}`), and executes `terraform plan` or `terraform apply -auto-approve`.

---

### Configuring GitHub Repository Secrets

To enable the workflows, add the following 5 secrets in your GitHub Repository under **Settings -> Secrets and variables -> Actions**:

| Secret Name | Description | Example / Format |
| :--- | :--- | :--- |
| **`BACKEND_CNAME`** | Cloud Run backend service hostname (without `https://`) | `<service-id>.<region>.run.app` |
| **`GCP_PROJECT_ID`** | Your Google Cloud Project ID | `<your-project-id>` |
| **`GCP_SERVICE_ACCOUNT`** | Email of the deployment service account | `github-actions-bot@<your-project-id>.iam.gserviceaccount.com` |
| **`GCP_WORKLOAD_IDENTITY_PROVIDER`** | Full resource path of the WIF Provider | `projects/<project-number>/locations/global/workloadIdentityPools/github-actions-pool/providers/github-provider` |
| **`GCP_TFSTATE_BUCKET`** | Name of the GCS bucket storing Terraform state | `<your-project-id>-tfstate` |

---

### How to Retrieve Secret Values Using `gcloud` CLI

Run these commands in your local shell to get the exact values for your GitHub secrets:

```bash
# 1. Get GCP_PROJECT_ID
gcloud config get-value project

# 2. Get GCP_WORKLOAD_IDENTITY_PROVIDER
gcloud iam workload-identity-pools providers describe github-provider \
    --location="global" \
    --workload-identity-pool="github-actions-pool" \
    --format="value(name)"

# 3. Get GCP_SERVICE_ACCOUNT
gcloud iam service-accounts list \
    --filter="name:github-actions-bot" \
    --format="value(email)"

# 4. Get GCP_TFSTATE_BUCKET
# This is the bucket name created during setup: gs://<your-project-id>-tfstate
echo "<your-project-id>-tfstate"

# 5. Get BACKEND_CNAME
gcloud run services describe ezyshare-backend \
    --region=us-central1 \
    --format="value(status.url)" | sed 's|https://||'
```

