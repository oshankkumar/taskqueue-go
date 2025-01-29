local pendingQueueKey = KEYS[1]
local jobKey = KEYS[2]
local jobID = ARGV[1]
local newScore = tonumber(ARGV[2])

-- Use ZADD with the XX modifier to update the score only if the job exists
local updated = redis.call("ZADD", pendingQueueKey, "XX", "CH", newScore, jobID)
if updated == 0 then
    return { err = "Job ID not found in the pending queue" }
end

-- Update fields in the job's hash
for i = 3, #ARGV, 2 do
    local field = ARGV[i]
    local value = ARGV[i + 1]
    redis.call("HSET", jobKey, field, value)
end

return { ok = 'OK' }