local queueKey = KEYS[1]
local jobKey = KEYS[2]
local queueSet = KEYS[3]
local jobID = ARGV[1]
local score = tonumber(ARGV[2])
local queueName = ARGV[3]

-- ADD the job ID to sorted set (queue)
redis.call('ZADD', queueKey, score, jobID)

-- SET the queue in queue set
redis.call('SADD', queueSet, queueName)

-- SET the job details in a hash
for i = 4, #ARGV, 2 do
    redis.call('HSET', jobKey, ARGV[i], ARGV[i + 1])
end

return {ok='OK'}