<template>
  <q-page class="q-pa-sm">
    <div class="row q-col-gutter-sm q-pa-sm">
      <div class="col-lg-12 col-md-12 col-sm-12 col-xs-12 q-pa-sm">
        <queue-stats-cards :dead-queues="deadQueues" :pending-queues="pendingQueues"
                           :active-workers-count="activeWorkers.length"/>
      </div>
    </div>

    <div class="row q-col-gutter-sm q-pa-sm">
      <div class="col-lg-12 col-md-12 col-sm-12 col-xs-12 q-pa-sm">
        <job-processing-graph/>
      </div>
    </div>


    <div class="row q-col-gutter-sm q-pa-sm">
      <div class="col-lg-6 col-md-6 col-sm-12 col-xs-12 q-pa-sm">
        <q-card v-if="loadingPendingQueues">
          <q-skeleton></q-skeleton>
        </q-card>
        <queue-statistics v-else title="Pending Queue Stats" :queue-details="pendingQueues"/>
      </div>

      <div class="col-lg-6 col-md-6 col-sm-12 col-xs-12 q-pa-sm">
        <q-card v-if="loadingDeadQueues">
          <q-skeleton></q-skeleton>
        </q-card>
        <queue-statistics v-else title="Dead Queue Stats" :queue-details="deadQueues"/>
      </div>
    </div>

    <div class="row q-col-gutter-sm q-pa-sm">
      <div class="col-lg-12 col-md-12 col-sm-12 col-xs-12 q-pa-sm">
        <worker-list :workers="activeWorkers" @refresh="fetchActiveWorkers"/>
      </div>
    </div>

  </q-page>
</template>

<script>
import QueueStatsCards from "components/QueueStatsCards.vue";
import QueueStatistics from "components/QueueStatistics.vue";
import WorkerList from "components/WorkerList.vue";
import JobProcessingGraph from 'components/JobProcessingGraph.vue';

export default {
  name: 'IndexPage',
  components: {
    QueueStatistics,
    WorkerList,
    JobProcessingGraph,
    QueueStatsCards,
  },
  data() {
    return {
      loadingPendingQueues: false,
      loadingDeadQueues: false,
      pendingQueues: [],
      deadQueues: [],
      activeWorkers: [],
    }
  },
  mounted() {
    this.fetchPendingQueues();
    this.fetchDeadQueues();
    this.fetchActiveWorkers();
  },
  methods: {
    async fetchActiveWorkers() {
      try {
        const data = await this.$taskManagerClient.listActiveWorkers();
        this.activeWorkers = data.activeWorkers || [];
      } catch (error) {
        console.log(error);
        this.activeWorkers = [];
        this.$q.notify({
          type: 'negative',
          message: 'Failed to fetch active workers.',
          timeout: 3000,
          icon: 'warning',
        });
      }
    },
    async fetchPendingQueues() {
      try {
        this.loadingPendingQueues = true;
        const data = await this.$taskManagerClient.listPendingQueues();
        this.pendingQueues = data.queues || [];
      } catch (error) {
        this.pendingQueues = [];
        console.error(error);
        this.$q.notify({
          type: 'negative',
          message: 'Failed to fetch pending queues.',
          timeout: 3000,
          icon: 'warning',
        });
      } finally {
        this.loadingPendingQueues = false;
      }
    },
    async fetchDeadQueues() {
      try {
        this.loadingDeadQueues = true;
        const data = await this.$taskManagerClient.listDeadQueues();
        this.deadQueues = data.queues || [];
      } catch (error) {
        this.deadQueues = [];
        console.error(error);
        this.$q.notify({
          type: 'negative',
          message: 'Failed to fetch dead queues.',
          timeout: 3000,
          icon: 'warning',
        });
      } finally {
        this.loadingDeadQueues = false;
      }
    }
  },
};
</script>
