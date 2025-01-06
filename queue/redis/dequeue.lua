-- ARGV[1]: The current epoch time (e.g., curr_epoch_time)
-- ARGV[2]: The invisibility duration (e.g., time in seconds to add to the current time)
-- ARGV[3]: The count of jobs to dequeue (e.g., number of job IDs to fetch)
-- KEYS[1]: The sorted set key (e.g., queue key)

local curr_time = tonumber(ARGV[1])        -- Current epoch time
local invisibility_duration = tonumber(ARGV[2]) -- Invisibility duration in seconds
local count = tonumber(ARGV[3])           -- Number of jobs to dequeue
local queue_key = KEYS[1]                 -- The Redis key for the sorted set

-- Fetch the specified number of job IDs with the smallest scores <= curr_time
local job_ids = redis.call("ZRANGEBYSCORE", queue_key, "-inf", curr_time, "LIMIT", 0, count)

if #job_ids == 0 then
    -- No jobs available for processing
    return nil
end

-- Prepare arguments for a single ZADD call
local zadd_args = {}
local new_score = curr_time + invisibility_duration

for _, job_id in ipairs(job_ids) do
    table.insert(zadd_args, new_score)  -- Add the new score
    table.insert(zadd_args, job_id)     -- Add the job ID
end

-- Perform a single ZADD call to update scores
redis.call("ZADD", queue_key, "XX", unpack(zadd_args))

-- Return the fetched job IDs
return job_ids
