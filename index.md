[Installation instructions](https://github.com/manics/binderhub-container-registry-helper/tree/main#readme)

{% for chartmap in site.data.index.entries %}
## {{ chartmap[0] }}

| Version | Date | App version |
|---------|------|-------------|
  {%- assign sortedcharts = chartmap[1] | sort: 'created' | reverse -%}
  {%- for chart in sortedcharts -%}
    {%- unless chart.version contains "-" %}
| [{{ chart.version }}]({{ chart.urls[0] }}) | {{ chart.created | date_to_long_string }} | {{ chart.appVersion }} |
    {%- endunless -%}
  {%- endfor %}
{% endfor %}
