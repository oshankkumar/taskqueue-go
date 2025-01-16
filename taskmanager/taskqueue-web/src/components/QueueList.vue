<template>
  <div class="row q-col-gutter-sm">
    <div class="col-lg-12 col-md-12 col-sm-12 col-xs-12">

      <q-card class="shadow-2 rounded-xl full-width">

        <!-- Title Section -->
        <q-card-section class="text-h6 q-pb-none">
          <q-item>
            <q-item-section avatar class="">
              <q-icon :color="queueIconColor" :name="queueIcon" size="44px" />
            </q-item-section>

            <q-item-section>
              <q-item-label>
                <div class="text-h6">{{ title }}</div>
              </q-item-label>
              <q-item-label caption class="text-black">
                Monitoring Your {{ queueType }} Queues.
              </q-item-label>
            </q-item-section>
          </q-item>
        </q-card-section>

        <!-- Queue Table -->
        <q-card-section>
          <q-table
            :rows="queueData"
            :columns="tableColumns"
            row-key="name"
            :rows-per-page-options="[5, 10, 20, 50]"
            class="my-table full-width"
            flat
            :pagination="pagination"
            @request="onRequest"
          >
            <template v-slot:body="props">
              <q-tr
                :props="props"
                class="cursor-pointer hover:bg-grey-2"
                @click="navigateToQueue(props.row.name)"
              >
                <!-- Queue Name -->
                <q-td key="name" :props="props" class="text-left">
                  <q-item>
                    <q-item-section>
                  <span class="text-primary text-bold">
                    {{ props.row.name }}
                  </span>
                    </q-item-section>
                  </q-item>
                </q-td>

                <!-- Job Count -->
                <q-td key="jobCount" :props="props" class="text-right">
                  {{ props.row.jobCount }}
                </q-td>

                <!-- Status -->
                <q-td key="status" :props="props" class="text-left">
                  <q-chip
                    :color="props.row.status === 'Running' ? 'green' : 'red'"
                    text-color="white"
                    size="sm"
                    outline
                  >
                    {{ props.row.status }}
                  </q-chip>
                </q-td>
              </q-tr>
            </template>
          </q-table>
        </q-card-section>
      </q-card>
    </div>
  </div>
</template>

<script>
export default {
  name: 'QueueList',
  props: {
    queueData: {
      type: Array,
      required: true,
    },
    title: String,
    queueType: String,
  },
  data() {
    return {
      columns: [
        {name: 'name', label: 'Queue Name', align: 'left', field: 'name'},
        {name: 'jobCount', label: 'Job Count', align: 'right', field: 'jobCount'},
      ],
      pagination: {
        page: 1,
        rowsPerPage: 10
      }
    };
  },
  computed: {
    queueIconColor() {
      return this.queueType === 'Pending' ? 'secondary' : 'negative';
    },
    queueIcon() {
      return this.queueType === 'Pending' ? 'pending' : 'cancel'
    },
    tableColumns() {
      if (this.queueType === 'Pending') {
        return this.columns.concat([{name: 'status', label: 'Status', align: 'right', field: 'status'}]);
      }
      return this.columns;
    }
  },
  methods: {
    // Handle pagination request
    onRequest(props) {
      console.log(props);
      this.pagination.page = props.page;
      this.pagination.rowsPerPage = props.rowsPerPage;
    },
    // Navigate to specific queue route on row click
    navigateToQueue(queueName) {
      if (this.queueType === 'Pending') {
        this.$router.push(`/pending-queues/${queueName}`);
      } else if (this.queueType === 'Dead') {
        this.$router.push(`/dead-queues/${queueName}`);
      }
    }
  }
};
</script>

<style scoped>
.q-page {
  padding: 16px;
}

.full-width {
  width: 100%;
}

.my-table {
  margin-top: 16px;
}

.q-table__header {
  background-color: #f5f5f5;
}

.q-table__row {
  cursor: pointer;
  transition: background-color 0.3s ease;
}

.q-table__row:hover {
  background-color: #f5f5f5 !important;
}

.text-center {
  text-align: center;
}
</style>
