#! /bin/bash
# SPDX-License-Identifier: MIT

if [ -z "${BLACKDUCK_TOKEN}" ]; then
  echo "BLACKDUCK_TOKEN must be set" && exit 1
fi

if [ -z "${BLACKDUCK_URL}" ]; then
  echo "BLACKDUCK_URL must be set" && exit 1
fi

if [ -z "${BLACKDUCK_PROJECT_NAME}" ]; then
  echo "BLACKDUCK_PROJECT_NAME must be set" && exit 1
fi

if [ -z "${BLACKDUCK_SCAN_VERSION_NAME}" ]; then
  echo "BLACKDUCK_SCAN_VERSION_NAME must be set" && exit 1
fi

if [ -z "${MAX_RETRY_COUNT}" ]; then
  MAX_RETRY_COUNT=12
fi


echo "Get Bearer Token ..."
bearer_token=$(curl -s -S -X POST "${BLACKDUCK_URL}/api/tokens/authenticate" \
  -H "Authorization: token ${BLACKDUCK_TOKEN}" \
  -H "Accept: application/vnd.blackducksoftware.user-4+json" \
  | jq -rc '.bearerToken')
echo "Lookup Project ..."
encoded_project_name=$(jq -rn --arg name "${BLACKDUCK_PROJECT_NAME}" '$name|@uri')
project_response=$(curl -s -S -X GET "${BLACKDUCK_URL}/api/projects?q=name:${encoded_project_name}" \
  -H "Authorization: Bearer ${bearer_token}" \
  -H "Accept: application/json" \
  -H "Content-Type: application/vnd.blackducksoftware.report-4+json")

project_count=$(echo "${project_response}" | jq '.totalCount')
project_url=""
if [ "${project_count}" -gt 0 ]; then
  project_url=$(echo "${project_response}" | jq -r --arg PROJECT_NAME "${BLACKDUCK_PROJECT_NAME}" '.items[] | select(.name==$PROJECT_NAME)._meta.href' | head -n 1)
  if [ -z "${project_url}" ]; then
    echo "No matching project with name ${BLACKDUCK_PROJECT_NAME} found."
    exit 1
  fi
else
  echo "Project lookup returns 0 items."
  exit 1
fi

echo "Lookup Version"
encoded_version_name=$(jq -rn --arg name "${BLACKDUCK_SCAN_VERSION_NAME}" '$name|@uri')
version_response=$(curl -s -S -X GET "${project_url}/versions?q=name:${encoded_version_name}" \
  -H "Authorization: Bearer ${bearer_token}" \
  -H "Accept: application/json" \
  -H "Content-Type: application/vnd.blackducksoftware.report-4+json")

version_count=$(echo "${version_response}" | jq '.totalCount')
version_links=""
if [ "${version_count}" -gt 0 ]; then
  version_links=$(echo "${version_response}" | jq -r --arg VERSION_NAME "${BLACKDUCK_SCAN_VERSION_NAME}" '.items[] | select(.versionName==$VERSION_NAME) | ._meta')
  if [ -z "${version_links}" ]; then
    echo "No matching project version with name ${BLACKDUCK_SCAN_VERSION_NAME} found."
    exit 1
  fi
else
  echo "Version lookup returns 0 items."
  exit 1
fi

echo "Get License Report URL ..."
license_report_url=$(echo ${version_links} | jq -r '.links[] | select(.rel=="licenseReports") | .href')

if [ -z "${license_report_url}" ]; then
  echo "License report URL could not be determined!"
  exit 1
fi
echo "License Report URL: ${license_report_url}"

echo "Trigger Report Creation ..."
report_create_response=$(curl -s -S -i -X POST ${license_report_url} \
  -H "Accept: */*" \
  -H "Authorization: Bearer ${bearer_token}" \
  -H "Content-Type: application/json" \
  -d '{"reportFormat":"TEXT","categories":["LICENSE_DATA","LICENSE_TEXT","COPYRIGHT_TEXT"]}')

# Check if the response is okay (200 or 201)
http_status=$(echo "${report_create_response}" | grep HTTP/ | tail -1 | awk '{print $2}')
echo "HTTP Status: ${http_status}"

if [ "${http_status}" -ne 200 ] && [ "${http_status}" -ne 201 ]; then
  echo "Failed to create Report, HTTP status: ${http_status}"
  exit 1
fi

# get report location
report_location=""
report_location=$(echo "${report_create_response}" | grep location | tail -1 | awk '{print $2}' | tr -d '\r')
if [ -z "${report_location}" ]; then
  echo "Unable to resolve Report location url from create report response"
fi
echo "Got Report Location URL: ${report_location}"

# check report status to be completed
retry_count=0
report_status=""

while [ "${report_status}" != "COMPLETED" ] && [ "${report_status}" != "FAILED" ] && [ ${retry_count} -lt ${MAX_RETRY_COUNT} ]; do
  sleep 10
  report_status=$(curl -s -S -X GET ${report_location} \
    -H "Accept: */*" -H "Authorization: Bearer ${bearer_token}" \
    | jq -r '.status')
  echo Retry ${retry_count}: Current report status is: ${report_status}
  let retry_count++ || true
done

if [ "${report_status}" == "FAILED" ]; then
  echo "Report creation failed after ${retry_count} retries!"
  exit 1
fi

if [ "${report_status}" != "COMPLETED" ]; then
  echo "Report creation is not finished after ${MAX_RETRY_COUNT} retries!"
  echo "Deleting stuck report..."
  curl -s -S -X DELETE ${report_location} \
    -H "Accept: */*" \
    -H "Authorization: Bearer ${bearer_token}"
  echo "Stuck report deleted."
  exit 1
fi

echo "Get URL for Report Download ..."
report_download_url=$(curl -s -S -X GET ${report_location} \
  -H "Accept: */*" \
  -H "Authorization: Bearer ${bearer_token}" \
  | jq -r '._meta.links[] | select(.rel=="download") | .href')

echo "Got Report Download URL: ${report_download_url}"

# download licenses.zip
curl -s -S -X GET ${report_download_url} \
  -H "Accept: */*" \
  -H "Authorization: Bearer ${bearer_token}" \
  -o ${BLACKDUCK_PROJECT_NAME}-licenses.zip

mv ${BLACKDUCK_PROJECT_NAME}-licenses.zip tmp/

unzip -j tmp/${BLACKDUCK_PROJECT_NAME}-licenses.zip

ls -l tmp/

mv tmp/version-license_*.txt tmp/Black_Duck_Notices_Report.txt
