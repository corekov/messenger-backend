$ErrorActionPreference = "Stop"

Write-Host "1. Registering user..."
$body = @{
    username = "testuser4"
    password = "testpassword123"
    device_name = "MyPhone"
    device_fp = "fp124"
    platform = "android"
} | ConvertTo-Json

$regResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/register" -Method Post -Body $body -ContentType "application/json"
$token = $regResponse.access_token
Write-Host "Token obtained: $token"

Write-Host "2. Getting profile (/auth/me)..."
$headers = @{ "Authorization" = "Bearer $token" }
$me = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/me" -Method Get -Headers $headers
$meJson = $me | ConvertTo-Json
Write-Host "Profile returned: $meJson"
$myId = $me.user_id
if (-not $myId) {
    # Fallbacks just in case
    $myId = $me.id
    if (-not $myId) {
        $myId = $me.user.id
    }
}

Write-Host "3. Creating a direct chat..."
$chatBody = @{
    type = "direct"
    member_ids = @($myId)
} | ConvertTo-Json
$chat = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/chats" -Method Post -Headers $headers -Body $chatBody -ContentType "application/json"
Write-Host "Chat created with ID: $($chat.id)"

Write-Host "4. Testing refresh token..."
$refreshBody = @{
    refresh_token = $regResponse.refresh_token
} | ConvertTo-Json

$refreshResp = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/refresh" -Method Post -Body $refreshBody -ContentType "application/json"
$newToken = $refreshResp.access_token
Write-Host "New token obtained: $newToken"

Write-Host "All basic endpoint verifications passed successfully!"
