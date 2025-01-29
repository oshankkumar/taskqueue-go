local jobKey = KEYS[1]
local pendingQueueKey = KEYS[2]
local deadQueueKey = KEYS[3]
local deadQueueSetKey = KEYS[4]
local jobID = ARGV[1]
local currentTime = tonumber(ARGV[2])
local deadQueueName = ARGV[3]

-- Remove the job ID from the pending queue
local removed = redis.call("ZREM", pendingQueueKey, jobID)
if removed == 0 then
    return { err = "Job ID not found in the pending queue" }
end

-- Add the job ID to the dead queue with the current timestamp as the score
redis.call("ZADD", deadQueueKey, currentTime, jobID)

-- Add the dead queue name to the set of dead queues
redis.call("SADD", deadQueueSetKey, deadQueueName)

-- Update fields in the job's hash
for i = 4, #ARGV, 2 do
    local field = ARGV[i]
    local value = ARGV[i + 1]
    redis.call("HSET", jobKey, field, value)
end

return { ok = 'OK' }
