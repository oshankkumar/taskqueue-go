local queueKey = KEYS[1]
local jobKey = KEYS[2]
local jobID = ARGV[1]
local jobTTLSec = tonumber(ARGV[2])

local removed = redis.call('ZREM', queueKey, jobID)
if removed == 0 then
    return { err = "Job ID not found in the pending queue" }
end

for i = 3, #ARGV, 2 do
    local field = ARGV[i]
    local value = ARGV[i + 1]
    redis.call('HSET', jobKey, field, value)
end

redis.call('EXPIRE', jobKey, jobTTLSec, 'NX')

return { ok = 'OK' }