import axios from 'axios';

class TaskManagerClient {
  constructor(baseURL = '/api') {
    this.api = axios.create({baseURL});
  }

  // Method to list all pending queues
  async listPendingQueues() {
    try {
      const response = await this.api.get('/pending-queues');
      return response.data;
    } catch (error) {
      console.error('Error fetching pending queues:', error);
      throw error;
    }
  }

  // Method to list all dead queues
  async listDeadQueues() {
    try {
      const response = await this.api.get('/dead-queues');
      return response.data;
    } catch (error) {
      console.error('Error fetching dead queues:', error);
      throw error;
    }
  }


  // Method to list all dead queues
  async listActiveWorkers() {
    try {
      const response = await this.api.get('/active-workers');
      return response.data;
    } catch (error) {
      console.error('Error fetching active workers:', error);
      throw error;
    }
  }

  // Method to list pending queue jobs with pagination (queue name, page number, rows count)
  async listPendingQueueJobs(queueName, page = 1, rows = 10) {
    try {
      const response = await this.api.get(`/pending-queues/${queueName}/jobs`, {
        params: {
          page,  // Page number
          rows,  // Number of rows per page
        },
      });
      return response.data;
    } catch (error) {
      console.error('Error fetching pending queue jobs:', error);
      throw error;
    }
  }

  // Method to list dead queue jobs with pagination (queue name, page number, rows count)
  async listDeadQueueJobs(queueName, page = 1, rows = 10) {
    try {
      const response = await this.api.get(`/dead-queues/${queueName}/jobs`, {
        params: {
          page,  // Page number
          rows,  // Number of rows per page
        },
      });
      return response.data;
    } catch (error) {
      console.error('Error fetching dead queue jobs:', error);
      throw error;
    }
  }

  // Method to create a new job by passing a JSON request body (createJobReq)
  async createJob(queueName, createJobReq) {
    try {
      const response = await this.api.post(`/pending-queues/${queueName}/jobs`, createJobReq);
      return response.data;
    } catch (error) {
      console.error('Error creating job:', error);
      throw error;
    }
  }

  // Method to toggle pending queue status
  async togglePendingQueueStatus(queueName) {
    try {
      const response = await this.api.post(`/pending-queues/${queueName}/toggle-status`);
      return response.data;
    } catch (error) {
      console.error('Error toggling pending queue status:', error);
      throw error;
    }
  }

  // Delete a job from a specific queue
  async deleteDeadJob(queueName, jobId) {
    const url = `/dead-queues/${queueName}/jobs/${jobId}`;
    try {
      const response = await this.api.delete(url);
      return response.data;
    } catch (error) {
      console.error(`Failed to delete job: ${error.message}`);
      throw error;
    }
  }

  // Delete all dead jobs from a specific queue
  async deleteAllDeadJobs(queueName) {
    const url = `/dead-queues/${queueName}/delete-all`;
    try {
      const response = await this.api.delete(url);
      return response.data;
    } catch (error) {
      console.error(`Failed to delete all dead jobs: ${error.message}`);
      throw error;
    }
  }


  // Requeue a job from a specific queue
  async requeueJob(queueName, jobId) {
    const url = `/dead-queues/${queueName}/jobs/${jobId}/requeue`;
    try {
      const response = await this.api.post(url);
      return response.data;
    } catch (error) {
      console.error(`Failed to requeue job: ${error.message}`);
      throw error;
    }
  }

  // Requeue all dead jobs from a specific queue
  async requeueAllJobs(queueName) {
    const url = `/dead-queues/${queueName}/requeue-all`;
    try {
      const response = await this.api.post(url);
      return response.data;
    } catch (error) {
      console.error(`Failed to requeue job: ${error.message}`);
      throw error;
    }
  }
}

export default TaskManagerClient;
