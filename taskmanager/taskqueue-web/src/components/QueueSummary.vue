<template>
  <q-card class="bg-transparent no-shadow no-border" bordered>
    <q-card-section class="q-pa-none">
      <div class="row q-col-gutter-sm ">
        <div v-for="(item, index) in items" :key="index" class="col-md-2-4 col-sm-6 col-xs-12">
          <q-item :style="`background-color: ${item.color}`" class="q-pa-none">
            <q-item-section class=" q-pa-md q-ml-none  text-white">
              <q-item-label class="text-white text-h6 text-weight-bolder">{{ item.value }}</q-item-label>
              <q-item-label>{{ item.title }}</q-item-label>
            </q-item-section>
            <q-item-section side class="q-mr-md text-white">
              <q-icon :name="item.icon" color="white" size="44px"></q-icon>
            </q-item-section>
          </q-item>
        </div>
      </div>
    </q-card-section>
  </q-card>
</template>

<script>

export default {
  name: "QueueSummary",
  props: {
    pendingQueues: {
      type: Array,
      required: true,
    },
    deadQueues: {
      type: Array,
      required: true,
    },
    activeWorkersCount: {
      type: Number,
      required: true,
    }
  },
  data() {
    return {}
  },
  computed: {
    items() {
      return [
        {
          title: "Active Workers",
          icon: "fas fa-cogs",
          value: this.workerCount,
          color: "#21ba45",
        },
        {
          title: "Total Queues",
          icon: "fas fa-stream",
          value: this.pendingQueuesCount,
          color: "#31ccec",
        },
        {
          title: "Pending Jobs",
          icon: "pending",
          value: this.pendingJobsCount,
          color: "#26a69a",
        },
        {
          title: "Dead Jobs",
          icon: "cancel",
          value: this.deadJobsCount,
          color: "#c10015",
        },
      ]
    },
    workerCount() {
      return this.activeWorkersCount.toString();
    },
    pendingQueuesCount() {
      return this.pendingQueues.length.toString();
    },
    pendingJobsCount() {
      return this.pendingQueues.reduce((total, queue) => total + queue.jobCount, 0).toString();
    },
    deadJobsCount() {
      return this.deadQueues.reduce((total, queue) => total + queue.jobCount, 0).toString();
    },
  },
}
</script>

<style scoped>
.col-md-2-4 {
  flex: 0 0 calc(100% / 4); /* Divide the row into 5 parts */
  max-width: calc(100% / 4); /* Ensure it fits in one row */
}
</style>
