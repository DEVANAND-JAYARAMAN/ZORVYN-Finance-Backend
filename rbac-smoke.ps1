$ErrorActionPreference = 'Stop'

function Invoke-Api {
  param(
    [string]$Method,
    [string]$Url,
    [hashtable]$Headers,
    $Body
  )

  try {
    if ($null -ne $Body) {
      $json = $Body | ConvertTo-Json -Depth 10
      $resp = Invoke-WebRequest -UseBasicParsing -Method $Method -Uri $Url -Headers $Headers -Body $json -ContentType 'application/json'
    } else {
      $resp = Invoke-WebRequest -UseBasicParsing -Method $Method -Uri $Url -Headers $Headers
    }

    return [pscustomobject]@{ StatusCode = [int]$resp.StatusCode; Body = $resp.Content }
  } catch {
    $status = 0
    $body = ''

    if ($_.Exception.Response) {
      $status = [int]$_.Exception.Response.StatusCode.value__
      $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
      $body = $reader.ReadToEnd()
      $reader.Close()
    } else {
      $body = $_.Exception.Message
    }

    return [pscustomobject]@{ StatusCode = $status; Body = $body }
  }
}

$base = 'http://localhost:8080/api/v1'
$results = New-Object System.Collections.Generic.List[object]

$creds = @(
  @{ Key='admin'; Email='admin@zorvyn.io'; Password='Admin@123' },
  @{ Key='analyst'; Email='analyst@zorvyn.io'; Password='Analyst@123' },
  @{ Key='viewer'; Email='viewer@zorvyn.io'; Password='Viewer@123' }
)

$tokens = @{}

foreach ($c in $creds) {
  $login = Invoke-Api -Method POST -Url "$base/auth/login" -Headers @{} -Body @{ email=$c.Email; password=$c.Password }
  $pass = $login.StatusCode -eq 200

  if ($pass) {
    $obj = $login.Body | ConvertFrom-Json
    $tokens[$c.Key] = $obj.token
  }

  $results.Add([pscustomobject]@{ Case="login_$($c.Key)"; Expected='200'; Got=[string]$login.StatusCode; Pass=$pass })
}

$adminH = @{ Authorization = "Bearer $($tokens['admin'])" }
$analystH = @{ Authorization = "Bearer $($tokens['analyst'])" }
$viewerH = @{ Authorization = "Bearer $($tokens['viewer'])" }

foreach ($role in @('admin','analyst','viewer')) {
  $h = if ($role -eq 'admin') { $adminH } elseif ($role -eq 'analyst') { $analystH } else { $viewerH }

  $me = Invoke-Api -Method GET -Url "$base/auth/me" -Headers $h -Body $null
  $results.Add([pscustomobject]@{ Case="me_$role"; Expected='200'; Got=[string]$me.StatusCode; Pass=($me.StatusCode -eq 200) })

  $sum = Invoke-Api -Method GET -Url "$base/dashboard/summary" -Headers $h -Body $null
  $results.Add([pscustomobject]@{ Case="dashboard_summary_$role"; Expected='200'; Got=[string]$sum.StatusCode; Pass=($sum.StatusCode -eq 200) })

  $week = Invoke-Api -Method GET -Url "$base/dashboard/weekly" -Headers $h -Body $null
  $results.Add([pscustomobject]@{ Case="dashboard_weekly_$role"; Expected='200'; Got=[string]$week.StatusCode; Pass=($week.StatusCode -eq 200) })
}

$recAdmin = Invoke-Api -Method GET -Url "$base/records" -Headers $adminH -Body $null
$results.Add([pscustomobject]@{ Case='records_list_admin'; Expected='200'; Got=[string]$recAdmin.StatusCode; Pass=($recAdmin.StatusCode -eq 200) })

$recAnalyst = Invoke-Api -Method GET -Url "$base/records" -Headers $analystH -Body $null
$results.Add([pscustomobject]@{ Case='records_list_analyst'; Expected='200'; Got=[string]$recAnalyst.StatusCode; Pass=($recAnalyst.StatusCode -eq 200) })

$recViewer = Invoke-Api -Method GET -Url "$base/records" -Headers $viewerH -Body $null
$results.Add([pscustomobject]@{ Case='records_list_viewer'; Expected='403'; Got=[string]$recViewer.StatusCode; Pass=($recViewer.StatusCode -eq 403) })

$payload = @{
  amount = 101.25
  type = 'expense'
  category = 'RBAC-Test'
  date = (Get-Date).ToUniversalTime().ToString('o')
  description = 'rbac smoke test record'
}

$createAnalyst = Invoke-Api -Method POST -Url "$base/records" -Headers $analystH -Body $payload
$results.Add([pscustomobject]@{ Case='records_create_analyst'; Expected='201'; Got=[string]$createAnalyst.StatusCode; Pass=($createAnalyst.StatusCode -eq 201) })

$recID = $null
if ($createAnalyst.StatusCode -eq 201) {
  $recID = ($createAnalyst.Body | ConvertFrom-Json).id
}

$createViewer = Invoke-Api -Method POST -Url "$base/records" -Headers $viewerH -Body $payload
$results.Add([pscustomobject]@{ Case='records_create_viewer'; Expected='403'; Got=[string]$createViewer.StatusCode; Pass=($createViewer.StatusCode -eq 403) })

if ($recID) {
  $updateAnalyst = Invoke-Api -Method PUT -Url "$base/records/$recID" -Headers $analystH -Body @{ description = 'updated by analyst' }
  $results.Add([pscustomobject]@{ Case='records_update_analyst'; Expected='200'; Got=[string]$updateAnalyst.StatusCode; Pass=($updateAnalyst.StatusCode -eq 200) })

  $deleteAnalyst = Invoke-Api -Method DELETE -Url "$base/records/$recID" -Headers $analystH -Body $null
  $results.Add([pscustomobject]@{ Case='records_delete_analyst'; Expected='403'; Got=[string]$deleteAnalyst.StatusCode; Pass=($deleteAnalyst.StatusCode -eq 403) })

  $deleteAdmin = Invoke-Api -Method DELETE -Url "$base/records/$recID" -Headers $adminH -Body $null
  $results.Add([pscustomobject]@{ Case='records_delete_admin'; Expected='204'; Got=[string]$deleteAdmin.StatusCode; Pass=($deleteAdmin.StatusCode -eq 204) })
}

$usersAdmin = Invoke-Api -Method GET -Url "$base/users" -Headers $adminH -Body $null
$results.Add([pscustomobject]@{ Case='users_list_admin'; Expected='200'; Got=[string]$usersAdmin.StatusCode; Pass=($usersAdmin.StatusCode -eq 200) })

$usersAnalyst = Invoke-Api -Method GET -Url "$base/users" -Headers $analystH -Body $null
$results.Add([pscustomobject]@{ Case='users_list_analyst'; Expected='403'; Got=[string]$usersAnalyst.StatusCode; Pass=($usersAnalyst.StatusCode -eq 403) })

$usersViewer = Invoke-Api -Method GET -Url "$base/users" -Headers $viewerH -Body $null
$results.Add([pscustomobject]@{ Case='users_list_viewer'; Expected='403'; Got=[string]$usersViewer.StatusCode; Pass=($usersViewer.StatusCode -eq 403) })

$logout = Invoke-Api -Method POST -Url "$base/auth/logout" -Headers $viewerH -Body $null
$results.Add([pscustomobject]@{ Case='logout_viewer'; Expected='200'; Got=[string]$logout.StatusCode; Pass=($logout.StatusCode -eq 200) })

$afterLogout = Invoke-Api -Method GET -Url "$base/auth/me" -Headers $viewerH -Body $null
$results.Add([pscustomobject]@{ Case='me_viewer_after_logout'; Expected='401'; Got=[string]$afterLogout.StatusCode; Pass=($afterLogout.StatusCode -eq 401) })

$failed = $results | Where-Object { -not $_.Pass }

Write-Output '==== RBAC_SMOKE_RESULTS ===='
$results | Sort-Object Case | Format-Table -AutoSize
Write-Output "TOTAL=$($results.Count) FAILED=$($failed.Count)"

if ($failed.Count -gt 0) {
  Write-Output '==== FAILED_CASES ===='
  $failed | Format-Table -AutoSize
  exit 1
}
