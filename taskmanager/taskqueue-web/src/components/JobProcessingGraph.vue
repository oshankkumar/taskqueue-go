<template>
  <q-card class="shadow-2 rounded-xl full-width">
    <q-card-section class="q-pb-none">
      <q-item>
        <q-item-section avatar class="">
          <q-icon color="blue" name="fas fa-chart-line" size="44px"/>
        </q-item-section>
        <q-item-section>
          <div class="text-h6">Job Processed</div>
        </q-item-section>
        <q-item-section side>
          <q-select
            v-model="selectedRange"
            :options="dateRangeOptions"
            label="Date Range"
            dense
            outlined
            class="q-mr-sm"
            @update:model-value="updateMetrics"
          />
        </q-item-section>
        <q-item-section side>
          <q-select
            v-model="selectedGranularity"
            :options="granularityOptions"
            label="Granularity"
            dense
            outlined
            @update:model-value="updateMetrics"
          />
        </q-item-section>
      </q-item>
    </q-card-section>
    <q-card-section>
      <v-chart :option="chartOptions" class="chart-container" autoresize/>
    </q-card-section>
  </q-card>
</template>

<script>
import {use} from "echarts/core";
import {CanvasRenderer} from "echarts/renderers";
import {LineChart} from "echarts/charts";
import {GridComponent, TitleComponent, TooltipComponent} from "echarts/components";
import VChart from "vue-echarts";

// Register ECharts components
use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, TitleComponent]);

export default {
  name: 'JobProcessingGraph',
  components: {
    VChart,
  },
  data() {
    return {
      chartOptions: {
        tooltip: {
          trigger: 'axis',
        },
        xAxis: {
          type: 'category',
          data: [],
          axisLabel: {
            formatter: (value) => value,
          },
        },
        yAxis: {
          type: 'value',
          name: 'Jobs Processed',
        },
        series: [
          {
            data: [],
            type: 'line',
            smooth: true,
            areaStyle: {
              color: 'rgba(66, 165, 245, 0.2)',
            },
            lineStyle: {
              color: '#42A5F5',
            },
          },
        ],
      },
      loading: true,
      dateRangeOptions: [
        {label: 'Last 1 Hour', value: '1h'},
        {label: 'Last 6 Hours', value: '6h'},
        {label: 'Last 24 Hours', value: '24h'},
        {label: 'Last 2 Days', value: '2d'},
      ],
      granularityOptions: [
        {label: '5 Minutes', value: '300'},
        {label: '15 Minutes', value: '900'},
        {label: '1 Hour', value: '3600'},
      ],
      selectedRange: {label: 'Last 24 Hours', value: '24h'},
      selectedGranularity: {label: '1 Hour', value: '3600'},
    };
  },
  methods: {
    async fetchMetrics(startTime, endTime, step) {
      this.loading = true;
      try {
        const {values} = await this.$taskManagerClient.fetchJobProcessedMetrics(startTime, endTime, step);
        const labels = values.map(item => new Date(item.timestamp * 1000).toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit'
        }));
        const data = values.map(item => item.value);

        this.chartOptions.xAxis.data = labels;
        this.chartOptions.series[0].data = data;
      } catch (error) {
        console.error('Error fetching metrics:', error);
      } finally {
        this.loading = false;
      }
    },
    updateMetrics() {
      const now = Math.floor(Date.now() / 1000); // Current time in seconds
      let startTime;
      const step = parseInt(this.selectedGranularity.value);

      switch (this.selectedRange.value) {
        case '1h':
          startTime = now - 3600; // Last 1 hour
          break;
        case '6h':
          startTime = now - 6 * 3600; // Last 6 hours
          break;
        case '24h':
          startTime = now - 24 * 3600; // Last 24 hours
          break;
        case '2d':
          startTime = now - 48 * 3600;
          break;
        default:
          startTime = now - 3600; // Default to last 1 hour
      }

      this.fetchMetrics(startTime, now, step);
    },
  },
  mounted() {
    this.updateMetrics();
  },
};
</script>

<style scoped>
.q-select {
  min-width: 150px;
}
</style>
