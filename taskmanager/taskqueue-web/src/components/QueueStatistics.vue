<template>
  <q-card class="shadow-2 rounded-xl full-width">
    <q-card-section class="q-pb-none">
      <q-item>
        <q-item-section avatar class="">
          <q-icon color="blue" name="fas fa-chart-line" size="44px"/>
        </q-item-section>
        <q-item-section>
          <div class="text-h6">{{ title}}</div>
        </q-item-section>
      </q-item>
    </q-card-section>
    <q-card-section>
      <v-chart :option="chartOptions" class="chart-container"/>
    </q-card-section>
  </q-card>
</template>

<script>
import {use} from "echarts/core";
import {CanvasRenderer} from "echarts/renderers";
import {BarChart} from "echarts/charts";
import {GridComponent, TitleComponent, TooltipComponent} from "echarts/components";
import VChart from "vue-echarts";

// Register ECharts components
use([CanvasRenderer, BarChart, GridComponent, TooltipComponent, TitleComponent]);

export default {
  name: "QueueStatsChart",
  components: {
    VChart,
  },
  props: {
    queueDetails: {
      type: Array,
      required: true,
      default: () => [],
    },
    title: String,
  },
  data() {
    return {
      chartOptions: {
        title: {
          // text: "Queue Statistics",
          left: "center",
        },
        tooltip: {
          trigger: "axis",
        },
        xAxis: {
          type: "category",
          data: [],
          axisLabel: {
            rotate: 30, // Rotate labels for better readability
          },
        },
        yAxis: {
          type: "value",
          name: "Job Count",
        },
        series: [
          {
            data: [],
            type: "bar",
            barWidth: "50%",
            itemStyle: {
              color: "#42A5F5",
            },
          },
        ],
      },
    };
  },
  watch: {
    queueDetails: {
      handler(newVal) {
        this.updateChartOptions(newVal);
      },
      immediate: true,
      deep: true,
    },
  },
  methods: {
    updateChartOptions(queueDetails) {
      const queueNames = queueDetails.map((item) => item.name);
      const jobCounts = queueDetails.map((item) => item.jobCount);

      this.chartOptions.xAxis.data = queueNames;
      this.chartOptions.series[0].data = jobCounts;
    },
  },
};
</script>

<style>
.chart-container {
  height: 400px;
  width: 100%;
}
</style>
