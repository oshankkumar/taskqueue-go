<template>
  <q-page class="q-pa-md">
    <div class="row q-col-gutter-sm q-py-sm">
      <div class="col-lg-6 col-md-6 col-sm-12 col-xs-12">
        <new-job-form :queues="availableQueues"/>
      </div>
    </div>
  </q-page>

</template>

<script>
import NewJobForm from 'components/NewJobForm.vue'

export default {
  name: 'SubmitJobPage',
  components: {
    NewJobForm,
  },
  data() {
    return {
      loadingPendingQueues: false,
      availableQueues: [],
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
        this.availableQueues = data.queues || [];
      } catch (error) {
        this.availableQueues = []
        console.error(error)
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
