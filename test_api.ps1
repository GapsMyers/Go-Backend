$ErrorActionPreference = "Stop"

$baseUrl = "http://localhost:8080/api"

Write-Host "1. Registering User..."
$uniqueStr = -join ((65..90) + (97..122) | Get-Random -Count 10 | % {[char]$_})
$email = "test_$uniqueStr@example.com"
$registerBody = @{
    name = "Test User"
    email = $email
    password = "password123"
} | ConvertTo-Json
$registerResponse = Invoke-RestMethod -Uri "$baseUrl/register" -Method Post -Body $registerBody -ContentType "application/json"
$token = $registerResponse.data.access_token
Write-Host "Registered and got Token: $token"

$headers = @{
    "Authorization" = "Bearer $token"
}

Write-Host "`n2. Creating Matkul..."
$matkulBody = @{
    name = "Pengembangan Perangkat Lunak"
    code = "IF3250"
    semester = "4"
    tag = "Core"
} | ConvertTo-Json
$matkulResponse = Invoke-RestMethod -Uri "$baseUrl/matkul" -Method Post -Headers $headers -Body $matkulBody -ContentType "application/json"
$matkulId = $matkulResponse.data.id
Write-Host "Created Matkul ID: $matkulId"

Write-Host "`n3. Creating Deadline..."
$dueAt = (Get-Date).AddDays(2).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$deadlineBody = @{
    matkul_id = $matkulId
    title = "Tugas 1"
    description = "Deskripsi tugas"
    due_at = $dueAt
    priority = 1
    reminder_offset_minutes = 60
} | ConvertTo-Json
$createDeadlineResponse = Invoke-RestMethod -Uri "$baseUrl/deadlines" -Method Post -Headers $headers -Body $deadlineBody -ContentType "application/json"
$deadlineId = $createDeadlineResponse.data.id
Write-Host "Created Deadline ID: $deadlineId"
Write-Host "Deadline Response Priority: $($createDeadlineResponse.data.priority) Reminder: $($createDeadlineResponse.data.reminder_offset_minutes)"

Write-Host "`n4. Listing Deadlines..."
$listDeadlinesResp = Invoke-RestMethod -Uri "$baseUrl/deadlines" -Method Get -Headers $headers
$count = $listDeadlinesResp.data.Length
Write-Host "Found $count Deadline(s)."

Write-Host "`n5. Updating Deadline..."
$updateDeadlineBody = @{
    priority = 2
    reminder_offset_minutes = 180
} | ConvertTo-Json
$updateDeadlineResp = Invoke-RestMethod -Uri "$baseUrl/deadlines/$deadlineId" -Method Patch -Headers $headers -Body $updateDeadlineBody -ContentType "application/json"
Write-Host "Updated Priority: $($updateDeadlineResp.data.priority) Reminder: $($updateDeadlineResp.data.reminder_offset_minutes)"

Write-Host "`n6. Toggling Status..."
$toggleResp = Invoke-RestMethod -Uri "$baseUrl/deadlines/$deadlineId/toggle" -Method Patch -Headers $headers
Write-Host "Toggled Status: $($toggleResp.data.status)"
$toggleResp2 = Invoke-RestMethod -Uri "$baseUrl/deadlines/$deadlineId/toggle" -Method Patch -Headers $headers
Write-Host "Toggled Status Again: $($toggleResp2.data.status)"

Write-Host "`n7. Deleting Deadline..."
Invoke-RestMethod -Uri "$baseUrl/deadlines/$deadlineId" -Method Delete -Headers $headers
Write-Host "Deleted Deadline."

Write-Host "`nAll verification steps completed successfully!"
