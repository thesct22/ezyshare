# Deployment Steps

This document serves as a reference on how to deploy the backend as a Google Cloud Run job and the frontend on GitHub Pages.

## Backend Deployment on Google Cloud Run

### Setup GCP Foundation

1. Go to the [Google Cloud Console](https://console.cloud.google.com/) and create/select a project.
2. Enable Billing for your project, if not already enabled. You will need to create a billing account if you don't have one. 
3. Enable APIs for terraform scripts to use. To do this, you need to open the cloud shell (>_ icon on the top right corner of cloud console). This will start a new bash session on the browser. On the terminal run the following command:
    ```bash
    gcloud services enable run.googleapis.com iam.googleapis.com cloudresourcemanager.googleapis.com
    ```

### Create Service Accounts

Terraform and GitHub actions will use this service account to deploy and manage resources on GCP.

1. In the search bar of the GCP Console, search for "Service Accounts" (under the "IAM & Admin" section)
2. Click on "Create Service Account"
3. Fill in the service account name (e.g. "github-actions-bot") and a description (e.g. "Service account for GitHub actions") and click "Create and Continue".
4. Grant it the following roles so it has the power to deploy and manage the resources we define in terraform:
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

