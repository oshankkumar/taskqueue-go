<template>
    <q-page class="q-pa-md">
        <div class="row q-col-gutter-sm q-py-sm">
            <div class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
                <queue-list :queue-data="deadQueueData" title="Dead" queue-type="Dead"/>
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
            deadQueueData: [
                {queueName: 'process-order-payment:dead', jobCount: 150},
                {queueName: 'process-order-completion:dead', jobCount: 75},
                {queueName: 'process-new-order:dead', jobCount: 200},
                {queueName: 'process-dispatch-order:dead', jobCount: 150},
                {queueName: 'process-order-cancellation:dead', jobCount: 75},
                {queueName: 'process-dispatch-order:dead', jobCount: 150},
                {queueName: 'process-order-cancellation:dead', jobCount: 75},
            ]
        };
    }
};
</script>
