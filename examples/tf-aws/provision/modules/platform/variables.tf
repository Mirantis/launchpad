
variable "platform_key" {
  description = "Platform name/label key for the platforms list, which must match the internal library of platforms."
  type        = string
}

variable "windows_password" {
  description = "Password to use with windows & winrm, which is used to generate the windows user_data."
  type        = string
  sensitive   = true
  default     = ""
}
