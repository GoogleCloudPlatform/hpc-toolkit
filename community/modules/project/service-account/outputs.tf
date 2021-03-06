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

output "email" {
  description = "Service account email (for single use)."
  value       = module.service_accounts.email
}
output "emails" {
  description = "Service account emails by name."
  value       = module.service_accounts.emails
}
output "emails_list" {
  description = "Service account emails s list."
  value       = module.service_accounts.emails_list
}
output "iam_email" {
  description = "IAM-format service account email (for single use)."
  value       = module.service_accounts.iam_email
}
output "iam_emails" {
  description = "IAM-format service account emails by name."
  value       = module.service_accounts.iam_emails
}
output "iam_emails_list" {
  description = "IAM-format service account emails s list."
  value       = module.service_accounts.iam_emails_list
}
output "key" {
  description = "Service account key (for single use)."
  value       = module.service_accounts.key
}
output "keys" {
  description = "Map of service account keys."
  value       = module.service_accounts.keys
}
output "service_account" {
  description = "Service account resource (for single use)."
  value       = module.service_accounts.service_account
}
output "service_accounts" {
  description = "Service account resources as list."
  value       = module.service_accounts.service_accounts
}
output "service_accounts_map" {
  description = "Service account resources by name."
  value       = module.service_accounts.service_accounts_map
}
