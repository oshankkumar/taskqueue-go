<template>
  <q-card class="shadow-2 rounded-xl full-width">
    <q-card-section>
      <q-item>
        <q-item-section>
          <q-item-label>
            <div class="text-h6">Active Workers</div>
          </q-item-label>
        </q-item-section>

        <q-item-section>
          <q-btn
            flat
            dense
            icon="refresh"
            label="Refresh"
            @click="$emit('refresh')"
            class="q-ml-auto"
            side
          />
        </q-item-section>
      </q-item>

    </q-card-section>

    <!-- Search and Filters -->


    <!-- Worker List -->
    <q-card-section>
      <q-virtual-scroll
        :items="workers"
        :items-per-page="pageSize"
        item-size="120"
        class="worker-list"
      >
        <template v-slot="{ item: worker }">
          <q-card flat bordered class="q-my-sm">
            <q-card-section>
              <q-item-label>
                <div class="text-bold">Worker ID: {{ worker.workerID }}</div>
              </q-item-label>
            </q-card-section>

            <q-item-section side class="q-mr-md text-white">
              <q-badge :color="worker.statusColor" label="Active" class="q-ml-auto"/>
            </q-item-section>

            <q-card-section>
              <div>Started At: {{ formatDate(worker.startedAt) }}</div>
              <div>Last Heartbeat: {{ formatDate(worker.heartbeatAt) }}</div>
              <div>Process ID: {{ worker.pid }}</div>
            </q-card-section>

            <q-card-section>
              <q-list bordered>
                <q-item v-for="queue in worker.queues" :key="queue.queueName">
                  <q-item-section>
                    <q-item-label>
                      <span class="text-bold">{{ queue.queueName }}</span>
                    </q-item-label>
                    <q-item-label caption class="text-grey-dark">
                      (Concurrency: {{ queue.concurrency }})
                    </q-item-label>
                  </q-item-section>
                </q-item>
              </q-list>
            </q-card-section>

          </q-card>
        </template>
      </q-virtual-scroll>
    </q-card-section>
  </q-card>
</template>

<script>
import {formatDistanceToNow, parseISO} from "date-fns";

export default {
  props: {
    workers: {type: Array, required: true},
    pageSize: {type: Number, default: 10},
  },
  data() {
    return {};
  },

  methods: {
    formatDate(date) {
      return formatDistanceToNow(parseISO(date), {addSuffix: true});
    },
  },
};
</script>

<style scoped>
.worker-list {
  max-height: 400px;
  overflow-y: auto;
}
</style>
