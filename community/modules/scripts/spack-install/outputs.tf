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

output "startup_script" {
  description = "Path to the Spack installation script."
  value       = local.script_content
}

output "controller_startup_script" {
  description = "Path to the Spack installation script, duplicate for SLURM controller."
  value       = local.script_content
}

output "install_spack_deps_runner" {
  description = <<-EOT
  Runner to install dependencies for spack using the startup-script module
  This runner requires ansible to be installed. This can be achieved using the
  install_ansible.sh script as a prior runner in the startup-script module:
  runners:
  - type: shell
    source: modules/startup-script/examples/install_ansible.sh
    destination: install_ansible.sh
  - $(spack.install_spack_deps_runner)
  ...
  EOT
  value       = local.install_spack_deps_runner
}

output "install_spack_runner" {
  description = "Runner to install Spack using the startup-script module"
  value       = local.install_spack_runner
}
