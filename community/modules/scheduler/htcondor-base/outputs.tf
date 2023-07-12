/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

output "access_point_service_account_email" {
  description = "HTCondor Access Point Service Account (e-mail format)"
  value       = module.access_point_service_account.email
  depends_on = [
    module.access_point_service_account
  ]
}

output "central_manager_service_account_email" {
  description = "HTCondor Central Manager Service Account (e-mail format)"
  value       = module.central_manager_service_account.email
  depends_on = [
    module.central_manager_service_account
  ]
}

output "execute_point_service_account_email" {
  description = "HTCondor Execute Point Service Account (e-mail format)"
  value       = module.execute_point_service_account.email
  depends_on = [
    module.execute_point_service_account
  ]
}

output "central_manager_runner" {
  description = "Toolkit Runner to configure an HTCondor Central Manager"
  value       = local.runner_cm
}

output "execute_point_runner" {
  description = "Toolkit Runner to configure an HTCondor Execute Point"
  value       = local.runner_execute
}

output "central_manager_ips" {
  description = "Reserved internal IP address for use by Central Manager"
  value       = module.address.addresses
}

output "windows_startup_ps1" {
  description = "Windows PowerShell script to update HTCondor configuration file"
  value       = local.windows_startup_ps1
}

output "htcondor_bucket_name" {
  description = "Name of the HTCondor configuration bucket"
  value       = module.htcondor_bucket.name
}
