local queueKey = KEYS[1]
local jobKeyPrefix = ARGV[1]
local maxScore = tonumber(ARGV[2]) -- Current epoch time
local invisibilityDuration = tonumber(ARGV[3])

local pauseKey = queueKey .. ':pause'
if redis.call('EXISTS', pauseKey) == 1 then
    return nil
end

-- Get the job with the lowest score in the queue
local jobIDs = redis.call("ZRANGE", queueKey, "-inf", maxScore, "BYSCORE", "LIMIT", 0, 1)
if #jobIDs == 0 then
    -- No jobs available for processing
    return nil
end

local jobID = jobIDs[1]

local newScore = maxScore + invisibilityDuration
-- Perform a single ZADD call to update scores
redis.call("ZADD", queueKey, "XX", newScore, jobID)

-- Fetch job details from the hash
local jobKey = jobKeyPrefix .. jobID
local jobDetails = redis.call("HGETALL", jobKey)

-- Return the job ID and its details
return { jobID, jobDetails }
