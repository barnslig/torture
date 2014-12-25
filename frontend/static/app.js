var Torture = (function() {
	"use strict";

	function Torture() {
		var self = this;

		self.tmpl = $('#tmpl-search-result').html();
		self.results = $('#search-results');
		self.form = $('#form-search');

		self.initEvents();
		Mustache.parse(self.tmpl);					
		self.retrieveResults(window.location.hash.slice(1));
	}
	(function(fn) {
		fn.initEvents = function() {
			var self = this;

			$(window).on('hashchange', function() {		
				self.retrieveResults(window.location.hash.slice(1));
			});
			self.form.on('submit', function(e) {
				e.preventDefault();
				window.location.hash = self.form.serialize();						
			});
		};

		fn.retrieveResults = function(query) {
			var self = this;

			self.results.empty();	
			$.ajax({
				url: self.form.attr('action'),
				data: query,
				dataType: 'json'
			}).done(function(data) {
				data.hits.forEach(function(hit, _) {
					self.results.append(Mustache.render(self.tmpl, {
						link: 'ftp://' + hit._source.Server + hit._source.Path,
						filename: hit._source.Path.split('/').pop(),
						filesize: filesize(hit._source.Size, {
							round: 0
						})
					}));
				});
			});
		};
	})(Torture.prototype);

	return Torture;

})();

$(function() {
	new Torture;
});
