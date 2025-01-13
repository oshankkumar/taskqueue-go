<template>
    <q-page class="q-pa-md">
        <div class="row q-col-gutter-sm q-py-sm">
            <div class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
                <queue-list :queue-data="pendingQueueData" title="Pending" queue-type="Pending"/>
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
            pendingQueueData: [
                {queueName: 'process-order-payment', jobCount: 150, status: 'Running'},
                {queueName: 'process-order-completion', jobCount: 75, status: 'Paused'},
                {queueName: 'process-new-order', jobCount: 200, status: 'Running'},
                {queueName: 'process-dispatch-order', jobCount: 150, status: 'Running'},
                {queueName: 'process-order-cancellation', jobCount: 75, status: 'Paused'},
                {queueName: 'process-dispatch-order', jobCount: 150, status: 'Running'},
                {queueName: 'process-order-cancellation', jobCount: 75, status: 'Paused'},
            ],
        };
    }
};
</script>
