<template>
  <q-card class="shadow-2 rounded-xl full-width">
    <q-card-section>
      <q-item>
        <q-item-section avatar class="">
          <q-icon color="blue" name="fas fa-cogs" size="44px"/>
        </q-item-section>
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
              <div class="row items-center">
                <div class="col">
                  <div class="text-h6">{{ worker.workerID }}</div>
                  <q-badge color="green" align="left">
                    Active
                  </q-badge>
                </div>
              </div>
            </q-card-section>

            <q-card-section>
              <p class="text-caption">
                <strong>Started At:</strong>
                {{ formatDate(worker.startedAt) }}
              </p>
              <p class="text-caption">
                <strong>Last Heartbeat:</strong>
                {{ formatDate(worker.heartbeatAt) }}
              </p>
            </q-card-section>

            <q-card-section>
              <q-expansion-item icon="queue" label="Queues">
                <q-list dense>
                  <q-item v-for="queue in worker.queues" :key="queue.name">
                    <q-item-section>
                      <span>{{ queue.queueName }}</span>
                    </q-item-section>
                    <q-item-section side>
                      <q-badge color="blue">{{ queue.concurrency }}</q-badge>
                    </q-item-section>
                  </q-item>
                </q-list>
              </q-expansion-item>
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
