<template>
    <q-page class="q-pa-md">
        <div class="row q-col-gutter-sm q-py-sm">
            <div v-if="loadingDeadQueues" class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
                <q-card>
                    <q-skeleton></q-skeleton>
                </q-card>
            </div>
            <div v-else class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
                <queue-list :queue-data="deadQueues" title="Dead" queue-type="Dead"/>
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
            loadingDeadQueues: false,
            deadQueues: []
        };
    },
    mounted() {
        this.fetchDeadQueues();
    },
    methods: {
        async fetchDeadQueues() {
            try {
                this.loadingDeadQueues = true;
                const data = await this.$taskManagerClient.listDeadQueues();
                this.deadQueues = data.queues || [];
            } catch (error) {
                this.deadQueues = [];
                console.error(error)
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
    }
};
</script>
