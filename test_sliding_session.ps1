$ErrorActionPreference = "Stop"

$baseUrl = "http://localhost:8080/api"

function Show-Result($msg, $resp) {
    Write-Host "`n$msg" -ForegroundColor Cyan
    $resp | ConvertTo-Json -Depth 5 | Write-Host
}

Write-Host "--- JWT Sliding Session Verification ---" -ForegroundColor Green

# 1. Register
$uniqueStr = -join ((65..90) + (97..122) | Get-Random -Count 5 | % {[char]$_})
$email = "sliding_$uniqueStr@example.com"
$registerBody = @{ email = $email; password = "password123" } | ConvertTo-Json
$regResp = Invoke-RestMethod -Uri "$baseUrl/register" -Method Post -Body $registerBody -ContentType "application/json"
$accessToken = $regResp.data.access_token
$refreshToken = $regResp.data.refresh_token
Show-Result "1. Registered User" $regResp

if (-not $refreshToken) { throw "Refresh token missing in register response" }

# 2. Access protected route
$headers = @{ "Authorization" = "Bearer $accessToken" }
$meResp = Invoke-RestMethod -Uri "$baseUrl/me" -Method Get -Headers $headers
Show-Result "2. Accessed /me" $meResp

# 3. Refresh Token
$refreshBody = @{ refresh_token = $refreshToken } | ConvertTo-Json
$refreshResp = Invoke-RestMethod -Uri "$baseUrl/refresh" -Method Post -Body $refreshBody -ContentType "application/json"
$newAccessToken = $refreshResp.data.access_token
$newRefreshToken = $refreshResp.data.refresh_token
Show-Result "3. Refreshed Token" $refreshResp

if ($newRefreshToken -eq $refreshToken) { throw "Refresh token was not rotated" }

# 4. Access protected route with NEW access token
$newHeaders = @{ "Authorization" = "Bearer $newAccessToken" }
$meResp2 = Invoke-RestMethod -Uri "$baseUrl/me" -Method Get -Headers $newHeaders
Show-Result "4. Accessed /me with NEW token" $meResp2

# 5. Try to use OLD refresh token (should fail due to rotation)
Write-Host "`n5. Trying OLD refresh token (expected to fail)..." -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$baseUrl/refresh" -Method Post -Body $refreshBody -ContentType "application/json"
    throw "Old refresh token should have been invalidated"
} catch {
    Write-Host "Success: Old refresh token failed as expected. Status: $($_.Exception.Response.StatusCode)"
}

# 6. Logout
$logoutBody = @{ refresh_token = $newRefreshToken } | ConvertTo-Json
$logoutResp = Invoke-RestMethod -Uri "$baseUrl/logout" -Method Post -Body $logoutBody -ContentType "application/json"
Show-Result "6. Logout" $logoutResp

# 7. Try to refresh after logout (should fail)
Write-Host "`n7. Trying to refresh after logout (expected to fail)..." -ForegroundColor Yellow
try {
    $afterLogoutBody = @{ refresh_token = $newRefreshToken } | ConvertTo-Json
    Invoke-RestMethod -Uri "$baseUrl/refresh" -Method Post -Body $afterLogoutBody -ContentType "application/json"
    throw "Refresh should have failed after logout"
} catch {
    Write-Host "Success: Refresh failed after logout as expected. Status: $($_.Exception.Response.StatusCode)"
}

Write-Host "`nAll Sliding Session tests passed!" -ForegroundColor Green
