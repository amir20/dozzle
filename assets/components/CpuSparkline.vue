<template></template>

<script lang="ts" setup>
import { select } from "d3-selection";
import { active, transition } from "d3-transition";
import { scaleLinear } from "d3-scale";
import { line } from "d3-shape";
import { easeLinear } from "d3-ease";

const d3 = { select, active, transition, scaleLinear, line, easeLinear };
console.log(d3);
</script>
