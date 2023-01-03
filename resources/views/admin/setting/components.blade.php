@extends('app')

@section('title','部件管理')

@section('content')
   <div class="row row-cards">
       @if($page->count())
           @foreach($page as $item)
               <div class="col-lg-4">
                   <div class="card-tabs">
                       <!-- Cards navigation -->
                       <ul class="nav nav-tabs">
                           <li class="nav-item"><a href="#tab-{{$item['id']}}-1" class="nav-link active" data-bs-toggle="tab">渲染结果</a></li>
                           <li class="nav-item"><a href="#tab-{{$item['id']}}-2" class="nav-link" data-bs-toggle="tab">引用代码</a></li>
                           <li class="nav-item"><a href="#tab-{{$item['id']}}-3" class="nav-link" data-bs-toggle="tab">文件路径</a></li>
                       </ul>
                       <div class="tab-content">
                           <!-- Content of card #1 -->
                           <div id="tab-{{$item['id']}}-1" class="card tab-pane active show">
                               <div class="card-body">
                                   @include($item['view'])
                               </div>
                           </div>
                           <!-- Content of card #2 -->
                           <div id="tab-{{$item['id']}}-2" class="card tab-pane">
                               <div class="card-body">
                                   <div class="card-title">引用代码</div>
                                   <p class="text-muted">
                                       <kbd>{{"<<"}}{{$item['import']}}{{">>"}}</kbd>
                                   </p>
                               </div>
                           </div>
                           <!-- Content of card #3 -->
                           <div id="tab-{{$item['id']}}-3" class="card tab-pane">
                               <div class="card-body">
                                   <div class="card-title">文件路径</div>
                                   <p class="text-muted">
                                       {{$item['path']}}
                                   </p>
                               </div>
                           </div>
                       </div>
                   </div>
               </div>
           @endforeach
           <div class="col-12">
               {!! make_page($page) !!}
           </div>
       @else
           <div class="col-lg-12">
               <div class="card card-body empty">
                   <div class="empty-header">403</div>
                   <p class="empty-title">暂无可用部件</p>
                   <p class="empty-subtitle text-muted">
                       如果你是开发者，可以在 {{BASE_PATH."/resources/views/customize/component/"}} 目录下创建
                   </p>
               </div>
           </div>
       @endif
   </div>
@endsection