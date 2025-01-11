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
      availableQueues: [
        {name: "Queue1"},
        {name: "Queue2"},
        {name: "Queue3"}
      ],
    };
  },
};
</script>
