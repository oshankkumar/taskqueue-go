<template>
  <q-page class="q-pa-md">
    <queue-summary/>

    <div class="row q-col-gutter-sm q-py-sm">
      <div class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
        <queue-statistics title="Pending Queue Stats" :queue-details="queueDetails"/>
      </div>
      <div class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
        <queue-statistics title="Dead Queue Stats" :queue-details="queueDetails"/>
      </div>
    </div>

  </q-page>
</template>

<script>
import QueueSummary from "components/QueueSummary.vue";
import QueueStatistics from "components/QueueStatistics.vue";

export default {
  name: 'IndexPage',
  components: {
    QueueSummary,
    QueueStatistics,
  },
  data() {
    return {
      queueDetails: [
        {queue_name: 'process-order-payment', job_count: 150},
        {queue_name: 'process-order-completion', job_count: 75},
        {queue_name: 'process-new-order', job_count: 200},
        {queue_name: 'process-dispatch-order', job_count: 150},
        {queue_name: 'process-order-cancellation', job_count: 75},
        {queue_name: 'process-dispatch-order', job_count: 150},
        {queue_name: 'process-order-cancellation', job_count: 75},
      ],
    }
  },
  computed: {},
  methods: {},
};
</script>
