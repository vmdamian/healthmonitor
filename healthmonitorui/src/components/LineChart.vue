
<script>
import { Line } from 'vue-chartjs'
export default {
	extends: Line,
	props: {
		label: {
			type: String,
			required: false,
		},
		data: {
			type: Array,
			required: false
		},
		labels: {
			type: Array,
			required: true
		}
	},
	data () {
		return {
			options: {
				scales: {
					yAxes: [{
						ticks: {
							beginAtZero: true
						},
						gridLines: {
							display: true
						}
					}],
					xAxes: [{
						gridLines: {
							display: false
						}
					}]
				},
				legend: {
					display: false
				},
				responsive: true,
				maintainAspectRatio: false
			}
		}
	},
	computed: {
		chartData: function() {
			return this.data;
		},
		chartLabels: function() {
			return this.labels;
		}
	},
	watch: {
		data: function() {
			this.renderChart({
				labels: this.chartLabels,
				datasets: [
				{
					label: this.label,
					borderColor: 'black',
					pointBackgroundColor: 'red',
					borderWidth: 1,
					pointBorderColor: '#249EBF',
					backgroundColor: 'transparent',
					data: this.chartData,
				}]
			}, this.options)
		}
	}
}
</script>