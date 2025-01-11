<template>
  <q-page class="q-pa-sm">
    <queue-list :queue-data="queueData" title="Pending Queues"/>
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
      queueData: [
        {queueName: 'process-order-payment', jobCount: 150, status: 'Running'},
        {queueName: 'process-order-completion', jobCount: 75, status: 'Paused'},
        {queueName: 'process-new-order', jobCount: 200, status: 'Running'},
        {queueName: 'process-dispatch-order', jobCount: 150, status: 'Running'},
        {queueName: 'process-order-cancellation', jobCount: 75, status: 'Paused'},
        {queueName: 'process-dispatch-order', jobCount: 150, status: 'Running'},
        {queueName: 'process-order-cancellation', jobCount: 75, status: 'Paused'},
      ]
    };
  }
};
</script>
