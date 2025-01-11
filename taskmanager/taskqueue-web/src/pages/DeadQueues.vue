<template>
  <q-page class="q-pa-sm">
    <queue-list :queue-data="queueData" title="Dead Queues"/>
  </q-page>
</template>

<script>
import QueueList from 'components/QueueList.vue'; // Adjust path as needed

export default {
  name: 'QueuePage',
  components: {
    QueueList
  },
  data() {
    return {
      // Example queue data
      queueData: [
        {queueName: 'process-order-payment:dead', jobCount: 15, status: 'Paused'},
        {queueName: 'process-order-completion:dead', jobCount: 7, status: 'Running'},
        {queueName: 'process-new-order:dead', jobCount: 20, status: 'Running'},
        {queueName: 'process-dispatch-order:dead', jobCount: 10, status: 'Running'},
        {queueName: 'process-order-cancellation:dead', jobCount: 5, status: 'Paused'},
      ]
    };
  }
};
</script>
