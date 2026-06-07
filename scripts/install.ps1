param(
  [string]$Version = $env:JIRA_CLI_VERSION,
  [string]$InstallDir = $env:JIRA_CLI_INSTALL_DIR,
  [switch]$AddToPath
)

$ErrorActionPreference = "Stop"

$Repo = if ($env:JIRA_CLI_REPO) { $env:JIRA_CLI_REPO } else { "sean2077/jira-cli" }
$Token = if ($env:GH_TOKEN) { $env:GH_TOKEN } else { $env:GITHUB_TOKEN }
$Headers = @{}
if ($Token) {
  $Headers["Authorization"] = "Bearer $Token"
}
if (-not $Version) {
  $Version = "latest"
}
if (-not $InstallDir) {
  $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\jira-cli"
}

function Resolve-LatestVersion {
  $location = $null
  try {
    Invoke-WebRequest -Uri "https://github.com/$Repo/releases/latest" -MaximumRedirection 0 -ErrorAction Stop | Out-Null
  } catch {
    if ($_.Exception.Response -and $_.Exception.Response.Headers.Location) {
      $location = $_.Exception.Response.Headers.Location.ToString()
    }
  }

  if ($location -and $location.Contains("/releases/tag/")) {
    return (($location -split "/releases/tag/")[-1] -replace "[?#].*$", "")
  }

  $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers $Headers
  return $release.tag_name
}

if ($Version -eq "latest") {
  $Version = Resolve-LatestVersion
}

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64"; break }
  "ARM64" { "arm64"; break }
  default {
    if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() -eq "Arm64") {
      "arm64"
    } else {
      throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
    }
  }
}

$asset = "jira_${Version}_windows_${arch}.exe"
$url = "https://github.com/$Repo/releases/download/$Version/$asset"
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) $asset
$target = Join-Path $InstallDir "jira.exe"

Write-Host "Downloading $url"
try {
  if ($Token) {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/tags/$Version" -Headers $Headers
    $assetInfo = $release.assets | Where-Object { $_.name -eq $asset } | Select-Object -First 1
    if (-not $assetInfo) {
      throw "Could not find release asset $asset in $Repo@$Version"
    }
    $downloadHeaders = @{} + $Headers
    $downloadHeaders["Accept"] = "application/octet-stream"
    Invoke-WebRequest -Uri $assetInfo.url -OutFile $tmp -Headers $downloadHeaders
  } else {
    Invoke-WebRequest -Uri $url -OutFile $tmp
  }
} catch {
  throw "Download failed. If $Repo is private, set GH_TOKEN or GITHUB_TOKEN with repo read access. $($_.Exception.Message)"
}
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Move-Item -Force $tmp $target

if ($AddToPath) {
  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  $paths = $userPath -split ";" | Where-Object { $_ }
  if ($paths -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable("Path", ($paths + $InstallDir -join ";"), "User")
    Write-Host "Added $InstallDir to the user PATH. Open a new terminal to use it."
  }
}

Write-Host "Installed $target"
& $target version

if (-not (($env:Path -split ";") -contains $InstallDir)) {
  Write-Host "NOTE $InstallDir is not on this terminal's PATH. Add it or open a new terminal after using -AddToPath."
}
