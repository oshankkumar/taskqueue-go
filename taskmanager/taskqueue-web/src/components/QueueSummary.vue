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
  data() {
    return {
      queuesCount: 10,
      pendingJobsCount: 50,
      completedJobsCount: 60,
      deadlineJobsCount: 50,
      workerCount: 10,
    }
  },
  computed: {
    items() {
      return [
        {
          title: "Active Workers",
          icon: "fas fa-cogs",
          value: this.workerCount.toString(),
          color: "#2196F3",
        },
        {
          title: "Total Queues",
          icon: "fas fa-stream",
          value: this.queuesCount.toString(),
          color: "#31ccec",
        },
        {
          title: "Pending Jobs",
          icon: "pending",
          value: this.pendingJobsCount.toString(),
          color: "#26a69a",
        },
        {
          title: "Completed Jobs",
          icon: "check_circle",
          value: this.completedJobsCount.toString(),
          color: "#21ba45",
        },
        {
          title: "Dead Jobs",
          icon: "cancel",
          value: this.deadlineJobsCount.toString(),
          color: "#c10015",
        },
      ]
    },
  }
}
</script>

<style scoped>
.col-md-2-4 {
  flex: 0 0 calc(100% / 5); /* Divide the row into 5 parts */
  max-width: calc(100% / 5); /* Ensure it fits in one row */
}
</style>
