(function () {
	var t = localStorage.getItem('blik-theme'),
		isListing = document.getElementById('listing-single') !== null,
		themeBtn = document.querySelector('.theme-btn'),
		layoutBtn = document.querySelector('.layout-btn'),
		tocBtn = document.querySelector('.toc-btn');

	if (!t) {
		t = window.matchMedia('(prefers-color-scheme:dark)').matches ? 'dark' : 'light';
	}
	document.documentElement.setAttribute('data-theme', t);
	if (themeBtn) themeBtn.textContent = t === 'dark' ? 'Light' : 'Dark';

	var initLayout;
	if (isListing) {
		initLayout = localStorage.getItem('blik-layout') || document.body.getAttribute('data-layout') || 'single';
		setListingLayout(initLayout);
	} else {
		initLayout = localStorage.getItem('blik-mdlayout') || document.body.getAttribute('data-layout') || 'single';
		setMdLayout(initLayout);
	}
	if (tocBtn) {
		tocBtn.addEventListener('click', function () {
			this.parentElement.classList.toggle('open');
		});
	}

	function setListingLayout(l) {
		var sections = ['listing-single', 'listing-dual', 'listing-triple'];
		sections.forEach(function (id) {
			var el = document.getElementById(id);
			if (el) el.classList.add('hidden');
		});
		if (l === 'dual') {
			var dual = document.getElementById('listing-dual');
			if (dual) dual.classList.remove('hidden');
			if (layoutBtn) layoutBtn.textContent = 'Dual';
		} else if (l === 'triple') {
			var triple = document.getElementById('listing-triple');
			if (triple) triple.classList.remove('hidden');
			if (layoutBtn) layoutBtn.textContent = 'Triple';
		} else {
			var single = document.getElementById('listing-single');
			if (single) single.classList.remove('hidden');
			if (layoutBtn) layoutBtn.textContent = 'Single';
		}
		localStorage.setItem('blik-layout', l);
	}

	function setMdLayout(l) {
		var c = document.getElementById('md-content');
		if (!c) return;
		if (l === 'dual') {
			c.classList.add('cols-dual');
			if (layoutBtn) layoutBtn.textContent = 'Dual';
		} else {
			c.classList.remove('cols-dual');
			if (layoutBtn) layoutBtn.textContent = 'Single';
		}
		localStorage.setItem('blik-mdlayout', l);
	}

	if (layoutBtn) {
		layoutBtn.addEventListener('click', function () {
			if (isListing) {
				var cur = localStorage.getItem('blik-layout') || 'single';
				var next = cur === 'single' ? 'dual' : cur === 'dual' ? 'triple' : 'single';
				setListingLayout(next);
			} else {
				var cur = localStorage.getItem('blik-mdlayout') || 'single';
				var next = cur === 'single' ? 'dual' : 'single';
				setMdLayout(next);
			}
		});
	}

	if (themeBtn) {
		themeBtn.addEventListener('click', function () {
			var h = document.documentElement;
			var t = h.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
			h.setAttribute('data-theme', t);
			localStorage.setItem('blik-theme', t);
			themeBtn.textContent = t === 'dark' ? 'Light' : 'Dark';
		});
	}

	var dataTable = document.getElementById('data-table');
	if (dataTable) {
		var headers = dataTable.querySelectorAll('th');
		headers.forEach(function (th, i) {
			th.addEventListener('click', function () {
				sortTable(dataTable, i);
			});
		});
	}

	function sortTable(table, col) {
		var tbody = table.querySelector('tbody'),
			rows = Array.prototype.slice.call(tbody.querySelectorAll('tr')),
			asc = table.getAttribute('data-sort-col') !== String(col) || table.getAttribute('data-sort-asc') !== 'true';
		rows.sort(function (a, b) {
			var av = a.children[col].textContent.trim(),
				bv = b.children[col].textContent.trim(),
				an = parseFloat(av),
				bn = parseFloat(bv);
			if (!isNaN(an) && !isNaN(bn)) {
				return asc ? an - bn : bn - an;
			}
			return asc ? av.localeCompare(bv) : bv.localeCompare(av);
		});
		rows.forEach(function (row) { tbody.appendChild(row); });
		table.setAttribute('data-sort-col', col);
		table.setAttribute('data-sort-asc', asc);
	}
})();
