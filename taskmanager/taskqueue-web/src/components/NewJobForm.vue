<template>
  <div class="row q-col-gutter-sm">
    <div class="col-lg-12 col-md-12 col-sm-12 col-xs-12">
      <q-card class="shadow-2 rounded-xl">
        <!-- Card Header -->
        <q-card-section>
          <q-item>
            <q-item-section avatar class="">
              <q-icon color="blue" name="task" size="44px"/>
            </q-item-section>

            <q-item-section>
              <q-item-label>
                <div class="text-h6">Submit Job</div>
              </q-item-label>
              <q-item-label caption class="text-black">
                You can create a new job.
              </q-item-label>
            </q-item-section>
          </q-item>
        </q-card-section>

        <!-- Form Section -->
        <q-card-section>
          <q-form @submit="submitJob" @reset="resetForm" ref="jobForm">
            <!-- Queue Selector -->
            <q-select
              v-model="formData.queue"
              label="Target Queue"
              outlined
              :options="queueOptions"
              :rules="[val => !!val || 'Queue selection is required']"
              class="q-mb-md"
              emit-value
              map-options
            />

            <!-- Job Argument -->
            <q-input
              v-model="formData.argument"
              label="Argument (JSON)"
              outlined
              :rules="[val => isValidJSON(val) || 'Invalid JSON format']"
              type="textarea"
              autogrow
              class="q-mb-md"
            />

            <!-- Submit and Reset Buttons -->
            <div class="row justify-end q-gutter-sm">
              <q-btn label="Reset" color="secondary" type="reset"/>
              <q-btn label="Submit" color="primary" type="submit" :loading="loading"/>
            </div>
          </q-form>
        </q-card-section>
      </q-card>
    </div>
  </div>
</template>

<script>
import axios from "axios";

export default {
  name: "NewJobForm",
  props: {
    queues: {
      type: Array,
      required: true,
      default: () => [],
    },
  },
  data() {
    return {
      formData: {
        queue: null,
        argument: "",
      },
      loading: false,
    };
  },
  computed: {
    queueOptions() {
      // Create options from the queues prop
      return this.queues.map((queue) => ({
        label: queue.name, // What the user sees
        value: queue.name, // The actual value bound to v-model
      }));
    },
  },
  methods: {
    async submitJob() {
      const form = this.$refs.jobForm;
      if (!form.validate()) {
        return;
      }

      this.loading = true;

      try {
        const payload = {
          queue: this.formData.queue,
          argument: JSON.parse(this.formData.argument),
        };

        await axios.post("/api/jobs", payload);
        this.$q.notify({type: "positive", message: "Job submitted successfully!"});
        this.resetForm();
      } catch (error) {
        console.error("Failed to submit job:", error);
        this.$q.notify({type: "negative", message: "Failed to submit the job."});
      } finally {
        this.loading = false;
      }
    },
    resetForm() {
      this.formData = {
        queue: null,
        argument: "",
      };
    },
    isValidJSON(value) {
      try {
        JSON.parse(value);
        return true;
      } catch {
        return false;
      }
    },
  },
};
</script>
