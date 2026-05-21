$ErrorActionPreference = "Stop"

Write-Host "1. Registering user..."
$body = @{
    username = "testuser11"
    password = "testpassword123"
    device_name = "MyPhone"
    device_fp = "fp128"
    platform = "android"
} | ConvertTo-Json

$regResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/register" -Method Post -Body $body -ContentType "application/json"
$token = $regResponse.access_token
Write-Host "Token obtained: $token"
Write-Host "Refresh token obtained: $($regResponse.refresh_token)"

Write-Host "4. Testing refresh token..."
$refreshBody = @{
    refresh_token = $regResponse.refresh_token
} | ConvertTo-Json

$refreshResp = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/refresh" -Method Post -Body $refreshBody -ContentType "application/json"
$newToken = $refreshResp.access_token
Write-Host "New token obtained: $newToken"

Write-Host "All basic endpoint verifications passed successfully!"
