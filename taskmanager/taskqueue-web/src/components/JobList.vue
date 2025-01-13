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

            <q-item-section v-if="queueType === 'Dead'" side>
              <q-btn
                round
                dense
                icon="play_arrow"
                color="primary"
                @click="reEnqueueAllJobs"
                aria-label="Re-enqueue All Jobs"
              >
                <q-tooltip>Re-enqueue All Jobs</q-tooltip>
              </q-btn>
            </q-item-section>

            <q-item-section v-if="queueType === 'Dead'" side>
              <q-btn
                round
                dense
                icon="delete_forever"
                color="negative"
                @click="deleteAllJobs"
                aria-label="Delete All Jobs"
              >
                <q-tooltip>Delete All Jobs</q-tooltip>
              </q-btn>
            </q-item-section>
          </q-item>


        </q-card-section>

        <q-card-section>
          <!-- Queue Table -->
          <q-table
            :rows="jobData"
            :columns="tableColumns"
            row-key="jobID"
            :pagination="pagination"
            :loading="loading"
            flat
            separator="horizontal"
            :rows-per-page-options="[5, 10, 20, 50]"
            :rows-per-page-label="'Rows per page'"
            class="my-table full-width"
            @request="onRequest"
          >
            <template v-slot:body="props">
              <q-tr :props="props">
                <!-- Job ID -->
                <q-td key="jobID" :props="props" class="text-left">
                  <q-item>
                    <q-item-section>
                      <span class="text-primary text-bold">{{ props.row.jobID }}</span>
                    </q-item-section>
                  </q-item>
                </q-td>

                <!-- Argument -->
                <q-td key="arg" :props="props" class="text-left">
                  <pre class="bg-grey-2 q-mb-none"
                       style="max-width: 400px; cursor: pointer; border-radius: 8px; font-family: monospace;"><code
                    class="json">{{ formatArgument(props.row.arg) }}</code></pre>
                </q-td>

                <!-- Started -->
                <q-td key="started" :props="props" class="text-left">
                  {{ formatRelativeTime(props.row.started) }}
                </q-td>

                <!-- Queued -->
                <q-td key="queued" :props="props" class="text-left">
                  {{ formatRelativeTime(props.row.queued) }}
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
                       style="max-width: 400px; cursor: pointer; border-radius: 8px; font-family: monospace;">
                    <code class="text-negative">{{ props.row.failureReason }}</code>
                  </pre>
                </q-td>

                <!-- Action Buttons -->
                <q-td key="action" :props="props" class="text-left">
                  <div class="q-gutter-sm flex justify-center">
                    <!-- Re-enqueue Button -->
                    <q-btn
                      round
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
                      round
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
import axios from "axios";
import {formatRelative} from 'date-fns';

const testData = [
  {
    jobID: "12345",
    arg: {name: 'Oshank', email: 'oshankfriends@gmail.com'},
    started: new Date(),
    queued: new Date(),
    status: 'Failed',
    failureReason: 'Error: Unable to establish a connection to the database. The request timed out after 30 seconds. Please check your database server configuration and ensure that it is reachable from the current network environment. Ensure that the database credentials provided are correct and that there are no firewall rules blocking access. If the issue persists, consult the system administrator or check the database logs for more detailed information about the failure.',
  },
  {
    jobID: "23456",
    arg: {
      id: 2937,
      name: "Item 72",
      isActive: true,
      price: "42.75",
      tags: "sale",
      details: {
        description: "This is a random description for item 42",
        dateAdded: "2025-01-13T12:34:56.789Z",
        quantity: 412
      },
      attributes: {
        weight: "2.53",
        dimensions: "24.51 x 35.12 x 47.73 cm"
      },
      customerReviews: [
        {
          reviewer: "User 54",
          rating: 4,
          comment: "This is a comment for item 62"
        },
        {
          reviewer: "User 21",
          rating: 5,
          comment: "This is another comment for item 17"
        }
      ]
    },
    started: new Date(),
    queued: new Date(),
    status: 'Waiting',
    failureReason: 'Error: File upload failed. The uploaded file exceeds the maximum allowed size of 10MB. Please reduce the size of the file and try again. If the issue persists, verify the file format and ensure that the file is not corrupted or damaged. If uploading multiple files, ensure that none of the files exceed the allowed limit. Please check your internet connection for any interruptions during the upload process.\n',
  },
  {
    jobID: "34567",
    arg: {name: 'Resham', email: 'resham1997@gmail.com'},
    started: new Date(),
    queued: new Date(),
    status: 'Active',
    failureReason: 'Error: Authentication failed due to invalid credentials. The username or password provided is incorrect. Please double-check the entered information and try again. If you have forgotten your password, use the "Forgot Password" option to reset it. Ensure that your account is active and has the appropriate permissions for accessing the requested resources. If you continue to encounter issues, please contact customer support for assistance.\n',
  },
  {
    jobID: "45678",
    arg: {name: 'Random', email: 'resham1997@gmail.com'},
    started: new Date(),
    queued: new Date(),
    status: 'Dead',
    failureReason: 'Error: API rate limit exceeded. You have made too many requests in a short period of time and have exceeded the allowed threshold. Please wait for 60 seconds before making additional requests. If you are using an automated system, consider implementing a delay or backoff strategy to prevent hitting the rate limits. For more information, refer to the API documentation for rate limiting details and best practices.\n'
  },
  {
    jobID: "56789",
    arg: {name: 'Allen', email: 'resham1997@gmail.com'},
    started: new Date(),
    queued: new Date(),
    status: 'Completed',
    failureReason: 'Error: You do not have permission to perform this action. The current user account is not authorized to access the requested resource or perform the desired operation. Please ensure that your user account has the necessary permissions assigned by the system administrator. If you believe this is an error, contact the administrator to request access or check the user roles and permissions configuration.\n',
  },
];

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
      menuModel: false,
      jobData: [],
      loading: false,
      columns: [
        {name: 'jobID', label: 'ID', align: 'left', field: 'jobID'},
        {name: 'arg', label: 'Argument', align: 'left', field: 'arg'},
        {name: 'started', label: 'Started', align: 'left', field: 'started'},
        {name: 'queued', label: 'Queued', align: 'left', field: 'Queued'},
      ],
      pagination: {page: 1, rowsPerPage: 10, rowsNumber: 0}
    };
  },
  computed: {
    queueType() {
      if (this.$route.path.startsWith('/dead-queues')) {
        return 'Dead';
      } else if (this.$route.path.startsWith('/complete-queues')) {
        return 'Completed';
      } else {
        return 'Pending';
      }
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
    this.jobData = testData;
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
      return formatRelative(new Date(timestamp), new Date());
    },

    onRequest(props) {
      console.log(props);

      if (!this.loading) {
        this.jobData = testData
        return;
      }
      // Set loading state
      this.loading = true;

      // Prepare query parameters
      const {page, rowsPerPage} = props.pagination;
      const offset = (page - 1) * rowsPerPage;

      // Fetch data from the backend
      axios.get("/api/jobs", {params: {offset, limit: rowsPerPage},})
        .then((response) => {
          // Update rows and pagination
          this.jobData = response.data.jobs; // Assuming API returns { jobs: [], total: number }
          this.pagination.rowsNumber = response.data.total; // Total rows from the backend
        })
        .catch((error) => {
          console.error("Failed to fetch jobs:", error);
        })
        .finally(() => {
          // Reset loading state
          this.loading = false;
        });
    },
    reEnqueueJob(job) {
      console.log(job);
    },
    deleteJob(job) {
      console.log(job);
    },
    reEnqueueAllJobs() {

    },
    deleteAllJobs() {

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
