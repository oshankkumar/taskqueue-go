<template>
    <q-page class="q-pa-md">
        <div class="row q-col-gutter-sm q-py-sm">
            <div v-if="loadingPendingQueues" class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
                <q-card>
                    <q-skeleton></q-skeleton>
                </q-card>
            </div>
            <div v-else class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
                <queue-list :queue-data="pendingQueues" title="Pending" queue-type="Pending"/>
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
            loadingPendingQueues: false,
            pendingQueues: [],
        };
    },
    mounted() {
        this.fetchPendingQueues();
    },
    methods: {
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
    }
};
</script>
