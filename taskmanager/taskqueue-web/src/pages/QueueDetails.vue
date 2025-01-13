<template>
  <q-page class="q-pa-md">
    <JobList :queueName="queueName"/>
  </q-page>
</template>

<script>
import JobList from "components/JobList.vue";

export default {
  name: "QueueDetailsPage",
  components: {
    JobList,
  },
  data() {
    return {
      queueName: this.$route.params.queue_name, // Extract queue name from route params
    };
  },
};
</script>

<style scoped>
.q-page {
  padding: 16px;
}

.shadow-2 {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}
</style>
