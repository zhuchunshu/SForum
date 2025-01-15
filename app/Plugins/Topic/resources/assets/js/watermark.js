/**
 * 给页面添加文字水印
 * @author  海角在眼前
 */
(function(){

  var watermark = function(self){
    this.elem = self;
  }

  watermark.prototype = {
    defaults : {
      texts : ['此处水印文字'],
      width : 100, //每行水印文字的水平间距
      height : 100, //水印文字的高度间距（低于文字高度会被替代）
      textRotate : -30 , //文字旋转 度数
      textColor : '#e5e5e5', //文字颜色
      textFont : '14px 微软雅黑' //字体
    },
    options : {
      canvas : []
    },
    init : function(options){
      $.extend(this.options, this.defaults, options);
      var $body = $('body'),
          can1 = this.__createCanvas($body),
          can2 = this.__createCanvas($body),
          canAll = this.__createCanvas($body),
          settings = this.options,
          txtlen = settings.texts.length;

      settings.deg = settings.textRotate * Math.PI / 180; //js里的正弦余弦用的是弧度

      this.__calcTextSize($body);
      var repeatTimes = Math.ceil(screen.width / settings.txts.length / settings.width);
      settings.canvasWidth = settings.canvasWidth * repeatTimes;
      var extTxts = [];
      while(repeatTimes--) extTxts = extTxts.concat(settings.txts);
      settings.txts = extTxts;

      var fixH = settings.maxWidth * Math.abs(Math.sin(settings.deg)) + Math.cos(settings.deg) * settings.textHeight;
      if(fixH > settings.height) settings.height = fixH;
      var ctx1 = this.__setCanvasStyle(can1, settings.canvasWidth, settings.height);
      var ctx2 = this.__setCanvasStyle(can2, settings.canvasWidth, settings.height);
      var ctx = this.__setCanvasStyle(canAll, settings.canvasWidth, settings.height * 2, true);

      this.__drawText(ctx1, settings.txts);
      this.__drawText(ctx2, settings.txts.reverse());

      //合并canvas
      ctx.drawImage(can1, 0, 0, settings.canvasWidth, settings.height);
      ctx.drawImage(can2, 0, settings.height, settings.canvasWidth, settings.height);
      var dataURL = canAll.toDataURL("image/png");
      $(this.elem).css('backgroundImage', "url("+ dataURL +")");
      //this.__destory();
    },
    __createCanvas : function($container){
      var canvas = document.createElement('canvas');
      $container.append(canvas);
      this.options.canvas.push(canvas);
      return canvas;
    },
    __calcTextSize : function($container){
      var txts = [],
          maxWidth = 0,
          canvasWidth = 0,
          settings = this.options;
      $.each(settings.texts, function(i, text){
        var span = $('<span style="font:'+settings.textFont+';visibility: hidden;display: inline-block;"> '+text+ '</span>')
            .appendTo($container);
        var tWidth = span[0].offsetWidth,
            tHeight = span[0].offsetHeight;
        span.remove();
        txts.push({
          txt : text,
          width : tWidth,
          height : tHeight
        });
        maxWidth = Math.max(maxWidth, tWidth);
        settings.textHeight = tHeight;
        var shadow = Math.cos(settings.deg) * tWidth;
        canvasWidth += (settings.width < shadow ? shadow : settings.width) - tHeight * Math.sin(settings.deg);
      });
      settings.txts = txts;
      settings.maxWidth = maxWidth;
      settings.canvasWidth = canvasWidth;
    },
    __setCanvasStyle : function(canvas, width, height, notextstyle){
      canvas.width = width;
      canvas.height = height;
      canvas.style.display='none';

      var ctx = canvas.getContext('2d');
      if(!notextstyle){
        var deg = this.options.deg,
            absSindeg = Math.abs(Math.sin(deg));
        ctx.rotate(deg);
        //基于视窗的 x、y偏移量
        var offset = absSindeg * this.options.height - this.options.textHeight * absSindeg;
        var nx = - offset * Math.cos(deg),
            ny = - offset * absSindeg;
        ctx.translate( nx, ny * absSindeg);

        ctx.font = this.options.textFont;
        ctx.fillStyle = this.options.textColor;
        ctx.textAlign = 'left';
        ctx.textBaseline = 'Middle';
      }
      return ctx;
    },
    __drawText: function(ctx, txts){
      var settings = this.options;
      $.each(txts, function(i, obj){

        var wnap = (settings.maxWidth - obj.width) / 2 ;
        var x = settings.width * Math.cos(settings.deg) * i,
            y = - x * Math.tan(settings.deg) + settings.height;
        ctx.fillText(obj.txt, x + wnap, y);
      });
    },
    __destory : function(){
      $.each(this.options.canvas, function(i, canvas){
        canvas.remove();
        canvas = null;
      });
    }
  }

  $.fn.watermark = function(options){
    new watermark(this).init(options);
  }

})(jQuery);
