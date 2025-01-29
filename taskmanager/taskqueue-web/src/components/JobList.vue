<template>
  <div class="row q-col-gutter-sm">
    <div class="col-lg-12 col-md-12 col-sm-12 col-xs-12">
      <!-- Title Section -->
      <q-card class="shadow-2 rounded-xl full-width">
        <!-- Title Section -->
        <q-card-section class="text-h6 q-pb-none">
          <q-item>
            <q-item-section avatar class="">
              <q-icon color="blue" name="fa fa-tasks" size="44px"/>
            </q-item-section>

            <q-item-section>
              <q-item-label>
                <div class="text-h6">{{ queueType }} Jobs</div>
              </q-item-label>
              <q-item-label caption class="text-black">
                Monitoring {{ queueName }} Jobs.
              </q-item-label>
            </q-item-section>

            <q-item-section v-if="queueType === 'Pending'" side>
              <q-btn
                :label="queueStatus === 'Running' ? 'Pause Queue' : 'Run Queue'"
                :icon="queueStatus === 'Running' ? 'pause' : 'play_arrow'"
                :color="queueStatus === 'Running' ? 'negative' : 'primary'"
                glossy
                rounded
                dense
                :loading="actionInProgress"
                @click="toggleQueueStatus"
                class="q-ml-auto q-px-md"
              />
            </q-item-section>

            <q-item-section v-if="queueType === 'Dead'" side>
              <q-btn
                label="Requeue All"
                icon="play_arrow"
                color="primary"
                glossy
                rounded
                dense
                @click="reEnqueueAllJobs"
                class="q-ml-auto q-px-md"
              />
            </q-item-section>

            <q-item-section v-if="queueType === 'Dead'" side>
              <q-btn
                label="Delete All"
                icon="delete_forever"
                color="negative"
                glossy
                rounded
                dense
                @click="deleteAllJobs"
                class="q-ml-auto q-px-md"
              />
            </q-item-section>
          </q-item>


        </q-card-section>

        <q-card-section>
          <!-- Queue Table  -->
          <q-table
            style="height: 700px"
            :rows="jobData"
            :columns="tableColumns"
            row-key="id"
            v-model:pagination="pagination"
            :loading="loading"
            flat
            separator="horizontal"
            virtual-scroll
            :rows-per-page-options="[10, 20, 50, 100]"
            :rows-per-page-label="'Rows per page'"
            class="my-table full-width "
            @request="handleJobListRequest"
          >
            <template v-slot:body="props">
              <q-tr :props="props">
                <!-- Job ID -->
                <q-td key="id" :props="props" class="text-left">
                  <q-item>
                    <q-item-section>
                      <span class="text-primary text-bold">{{ props.row.id }}</span>
                    </q-item-section>
                  </q-item>
                </q-td>

                <!-- Queued -->
                <q-td key="createdAt" :props="props" class="text-left">
                  {{ formatRelativeTime(props.row.createdAt) }}
                </q-td>

                <!-- Started -->
                <q-td key="startedAt" :props="props" class="text-left">
                  {{ formatRelativeTime(props.row.startedAt) }}
                </q-td>

                <!-- Argument -->
                <q-td key="args" :props="props" class="text-left">
                  <pre class="bg-grey-2 q-mb-none"
                       style="max-width: 400px; cursor: pointer; border-radius: 8px; font-family: monospace;"><code
                    class="json">{{ formatArgument(props.row.args) }}</code></pre>
                </q-td>

                <!-- Status -->
                <q-td key="status" :props="props" class="text-left">
                  <q-chip
                    :color="statusColor(props.row.status)"
                    text-color="white"
                    size="sm"
                    outline
                  >
                    {{ props.row.status }}
                  </q-chip>
                </q-td>

                <!-- Failure Reason -->
                <q-td key="failureReason" :props="props" class="text-left">
                  <q-tooltip>
                    {{ props.row.failureReason }}
                  </q-tooltip>
                  <pre class="bg-grey-2 q-mb-none"
                       style="max-width: 400px; cursor: pointer; border-radius: 8px; font-family: monospace;"><code
                    class="text-negative">{{ props.row.failureReason }}</code></pre>
                </q-td>

                <!-- Action Buttons -->
                <q-td key="action" :props="props" class="text-left">
                  <div class="q-gutter-sm flex justify-center">
                    <!-- Re-enqueue Button -->
                    <q-btn
                      glossy
                      rounded
                      dense
                      icon="play_arrow"
                      color="primary"
                      @click="reEnqueueJob(props.row)"
                      aria-label="Re-enqueue"
                      size="md"
                    >
                      <q-tooltip>Re-enqueue Job</q-tooltip>  <!-- Tooltip on Hover -->
                    </q-btn>

                    <!-- Delete Button with Tooltip -->
                    <q-btn
                      glossy
                      rounded
                      dense
                      icon="delete"
                      color="negative"
                      @click="deleteJob(props.row)"
                      aria-label="Delete"
                      size="md"
                    >
                      <q-tooltip>Delete Job</q-tooltip>  <!-- Tooltip on Hover -->
                    </q-btn>
                  </div>
                </q-td>

              </q-tr>
            </template>
          </q-table>
        </q-card-section>
      </q-card>
    </div>
  </div>
</template>

<script>
import {formatRelative} from 'date-fns';


export default {
  name: 'JobList',
  props: {
    queueName: {
      type: String,
      required: true,
    },
  },
  data() {
    return {
      queueStatus: 'Running',
      actionInProgress: false,
      menuModel: false,
      jobData: [],
      loading: false,
      columns: [
        {name: 'id', label: 'ID', align: 'left', field: 'id'},
        {name: 'createdAt', label: 'Queued', align: 'left', field: 'createdAt'},
        {name: 'startedAt', label: 'Started', align: 'left', field: 'startedAt'},
        {name: 'args', label: 'Argument', align: 'left', field: 'args'},
      ],
      pagination: {page: 1, rowsPerPage: 10, rowsNumber: 0}
    };
  },
  computed: {
    queueType() {
      if (this.$route.path.startsWith('/dead-queues')) {
        return 'Dead';
      }
      return 'Pending';
    },
    tableColumns() {
      if (this.queueType === 'Dead') {
        return this.columns.concat([
          {name: 'failureReason', label: 'Failure Reason', align: 'left', field: 'failureReason'},
          {name: 'action', label: 'Action', align: 'center', field: 'action'}
        ]);
      }
      return this.columns.concat([{name: 'status', label: 'Status', align: 'right', field: 'status'}]);
    }
  },
  mounted() {
    this.fetchJobs();
  },
  methods: {
    statusColor(status) {
      if (status === 'Waiting') {
        return 'blue-grey';
      }
      if (status === 'Active') {
        return 'primary';
      }
      if (status === 'Completed') {
        return 'positive';
      }
      if (status === 'Failed') {
        return 'warning';
      }
      if (status === 'Dead') {
        return 'negative';
      }
    },
    formatArgument(arg) {
      try {
        return JSON.stringify(arg, null, 2); // Pretty print JSON
      } catch (e) {
        console.error(e);
        return String(arg); // Fallback for invalid JSON
      }
    },

    formatRelativeTime(timestamp) {
      const date = new Date(timestamp);
      return date.toISOString() === "0001-01-01T00:00:00.000Z" || isNaN(date.getTime()) ? 'N/A' : formatRelative(new Date(timestamp), new Date());
    },

    async toggleQueueStatus() {
      this.actionInProgress = true;
      try {
        const newStatus = this.queueStatus === 'Running' ? 'Paused' : 'Running';
        const data = await this.$taskManagerClient.togglePendingQueueStatus(this.queueName);
        // Update the status on success
        this.queueStatus = data.newStatus;
        this.$q.notify({
          type: 'positive',
          message: `Queue is now ${newStatus}!`,
        });
      } catch (error) {
        console.error(error);
        this.$q.notify({
          type: 'negative',
          message: 'Failed to update queue status. Please try again.',
        });
      } finally {
        this.actionInProgress = false;
      }
    },

    async fetchJobs(page = 1, rowsPerPage = 10) {
      try {
        this.loading = true;
        const data = this.queueType === 'Dead' ?
          await this.$taskManagerClient.listDeadQueueJobs(this.queueName, page, rowsPerPage) :
          await this.$taskManagerClient.listPendingQueueJobs(this.queueName, page, rowsPerPage);

        this.jobData = data.jobs || [];
        this.queueStatus = data.status;
        this.pagination.page = page;
        this.pagination.rowsPerPage = rowsPerPage;
        this.pagination.rowsNumber = data.jobCount;
      } catch (error) {
        this.$q.notify({
          type: 'negative',
          message: 'Failed to fetch jobs.',
          timeout: 3000,
          icon: 'warning',
        });
        console.error(error)
      } finally {
        this.loading = false;
      }
    },

    async handleJobListRequest(props) {
      const {page, rowsPerPage} = props.pagination;
      await this.fetchJobs(page, rowsPerPage);
    },

    async reEnqueueJob(job) {
      try {
        await this.$taskManagerClient.requeueJob(this.queueName, job);
        await this.fetchJobs(this.pagination.page, this.pagination.rowsPerPage);
        this.$q.notify({type: "positive", message: "Job enqueued.", icon: "check_circle"});
      } catch (error) {
        this.$q.notify({type: "negative", message: "Failed to re-enqueue job.", icon: 'warning'});
        console.error(error);
      }
    },

    async deleteJob(job) {
      try {
        await this.$taskManagerClient.deleteDeadJob(this.queueName, job.id);
        await this.fetchJobs(this.pagination.page, this.pagination.rowsPerPage);
        this.$q.notify({type: "positive", message: "Job deleted successfully.", icon: "check_circle"});
      } catch (error) {
        this.$q.notify({type: "negative", message: "Failed to delete dead job."});
        console.error(error);
      }
    },

    async reEnqueueAllJobs() {
      try {
        await this.$taskManagerClient.requeueAllJobs(this.queueName);
        await this.fetchJobs(this.pagination.page, this.pagination.rowsPerPage);
        this.$q.notify({type: "positive", message: "All Jobs enqueued successfully.", icon: "check_circle"});
      } catch (error) {
        this.$q.notify({type: "negative", message: "Failed to enqueue jobs.", icon: "warning"});
        console.error(error);
      }
    },
    async deleteAllJobs() {
      try {
        await this.$taskManagerClient.deleteAllDeadJobs(this.queueName);
        await this.fetchJobs(this.pagination.page, this.pagination.rowsPerPage);
        this.$q.notify({type: "positive", message: "All jobs deleted successfully.", icon: "check_circle"});
      } catch (error) {
        this.$q.notify({type: "negative", message: "Failed to delete dead jobs.", icon: "warning"});
        console.error(error);
      }
    },
  }
};
</script>

<style scoped>
.q-page {
  padding: 16px;
}

.q-card {
  margin-top: 16px;
}

.q-table__row {
  transition: background-color 0.3s ease;
}

.q-table__row:hover {
  background-color: #f5f5f5 !important;
}

.text-primary {
  color: #1e88e5; /* Quasar's primary color */
}

pre {
  font-size: 0.9em;
  color: #333;
  background: #f8f8f8;
  padding: 8px;
  border-radius: 4px;
  overflow-x: auto;
}

.text-truncate {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

</style>
