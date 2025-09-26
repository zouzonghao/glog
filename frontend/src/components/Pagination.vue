<template>
  <nav v-if="totalPages > 1" class="pagination-new">
    <a href="#" @click.prevent="changePage(currentPage - 1)" :class="{ 'disabled': currentPage === 1 }" class="prev-next">上一页</a>

    <div class="pagination-center">
      <div class="page-numbers">
        <template v-for="page in pages" :key="page.number">
          <a v-if="page.isLink" href="#" @click.prevent="changePage(page.number)" class="page-number">{{ page.number }}</a>
          <span v-else-if="page.number" class="page-number current">{{ page.number }}</span>
          <span v-else class="ellipsis">…</span>
        </template>
      </div>

      <div v-if="pageSizeOptions && pageSizeOptions.length > 0" class="page-size-selector">
        <select :value="pageSize" @change="onPageSizeChange($event.target.value)">
          <option v-for="size in pageSizeOptions" :key="size" :value="size">
            {{ size }} / 页
          </option>
        </select>
      </div>
    </div>

    <a href="#" @click.prevent="changePage(currentPage + 1)" :class="{ 'disabled': currentPage === totalPages }" class="prev-next">下一页</a>
  </nav>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  currentPage: {
    type: Number,
    required: true,
  },
  totalPages: {
    type: Number,
    required: true,
  },
  pageSize: {
    type: Number,
    default: 10,
  },
  pageSizeOptions: {
    type: Array,
    default: () => [10, 20, 50],
  },
});

const emit = defineEmits(['page-changed', 'pagesize-changed']);

const changePage = (page) => {
  if (page > 0 && page <= props.totalPages && page !== props.currentPage) {
    emit('page-changed', page);
  }
};

const onPageSizeChange = (size) => {
  emit('pagesize-changed', parseInt(size, 10));
};

const pages = computed(() => {
  const total = props.totalPages;
  const current = props.currentPage;
  const delta = 2;
  const left = current - delta;
  const right = current + delta + 1;
  const range = [];
  const rangeWithDots = [];
  let l;

  for (let i = 1; i <= total; i++) {
    if (i === 1 || i === total || (i >= left && i < right)) {
      range.push({ number: i, isLink: i !== current });
    }
  }

  for (const i of range) {
    if (l) {
      if (i.number - l.number === 2) {
        rangeWithDots.push({ number: l.number + 1, isLink: true });
      } else if (i.number - l.number !== 1) {
        rangeWithDots.push({ number: null, isLink: false }); // Ellipsis
      }
    }
    rangeWithDots.push(i);
    l = i;
  }

  return rangeWithDots;
});
</script>

<style scoped>
/* Styles are based on the original project's style.css for pagination-new */
.pagination-new {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 2rem;
  user-select: none;
}

.pagination-center {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.page-numbers {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.page-number, .ellipsis {
  display: inline-block;
  min-width: 32px;
  height: 32px;
  line-height: 32px;
  text-align: center;
  border-radius: 4px;
}

.page-number {
  text-decoration: none;
  color: var(--text-color);
  background-color: var(--bg-secondary);
  transition: background-color 0.2s;
}

.page-number:hover {
  background-color: var(--primary-color);
  color: #fff;
}

.page-number.current {
  background-color: var(--primary-color);
  color: #fff;
  font-weight: bold;
}

.prev-next {
  text-decoration: none;
  color: var(--text-color);
  padding: 0 10px;
}

.prev-next.disabled {
  color: var(--text-muted);
  pointer-events: none;
}

.page-size-selector select {
  background-color: var(--bg-secondary);
  color: var(--text-color);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 5px;
}
</style>