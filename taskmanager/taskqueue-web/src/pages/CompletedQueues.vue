<template>
  <q-page class="q-pa-md">
    <div class="row q-col-gutter-sm q-py-sm">
      <div class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
        <queue-list :queue-data="completedQueueData" title="Completed" queue-type="Completed"/>
      </div>
    </div>
  </q-page>
</template>

<script>
import QueueList from 'components/QueueList.vue';

export default {
  name: 'QueuePage',
  components: {
    QueueList
  },
  data() {
    return {
      // Example queue data

      completedQueueData: [
        {queueName: 'process-order-payment:completed', jobCount: 150},
        {queueName: 'process-order-completion:completed', jobCount: 75},
        {queueName: 'process-new-order:completed', jobCount: 200},
        {queueName: 'process-dispatch-order:completed', jobCount: 150},
        {queueName: 'process-order-cancellation:completed', jobCount: 75},
        {queueName: 'process-dispatch-order:completed', jobCount: 150},
        {queueName: 'process-order-cancellation:completed', jobCount: 75},
      ],
    };
  }
};
</script>
