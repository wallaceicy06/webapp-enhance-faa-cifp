# Enhance FAA CIFP (the web app)

A web app for automatically processing the FAA CIFP data. It is under
construction and hosted at
[enhance-faa-cifp.seanharger.com](http://enhance-faa-cifp.seanharger.com).

![](https://github.com/wallaceicy06/webapp-enhance-faa-cifp/workflows/Go%20Tests/badge.svg)

## Run Locally

You can run the app locally by running the following commands.

1. Start the firestore emulator.

    ```shell
    gcloud beta emulators firestore start
    ```

1. Observe the output port number for the firestore emulator. Use the following
   flags to start the server.

   ```shell
   FIRESTORE_EMULATOR_HOST="localhost:$PORT" go run main.go --noauth --project_id="${PROJECT_ID}" \
   --gcs_bucket="${GCS_BUCKET}" --port=9999 --service_account_email=nobody
   ```