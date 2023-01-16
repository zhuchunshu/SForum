@extends('app')

@section('title','部件管理')

@section('content')
   <div class="row row-cards">
       <div class="col-lg-12">
           <div class="card">
               <div class="card-header">
                   <h3 class="card-title">部件列表</h3>
               </div>
               <div class="table-responsive">
                   <table class="table card-table table-vcenter">
                       <thead>
                       <tr>
                           <th>#</th>
                           <th>调用代码</th>
                           <th>备注</th>
                           <th></th>
                           <th></th>
                       </tr>
                       </thead>
                       <tbody>
                       @if(!$page->count())
                           <tr>
                               <td class="text-muted">暂无更多结果</td>
                               <td class="text-muted">暂无更多结果</td>
                               <td class="text-muted">暂无更多结果</td>
                               <td class="text-muted">暂无更多结果</td>
                           </tr>
                       @else
                           @foreach($page as $component)
                               <tr>
                                   <td class="text-muted">
                                       {{$component['id']}}
                                   </td>
                                   <td class="text-muted">
                                        {{$component['import']}}
                                   </td>
                                   <td class="text-muted">
                                        {{$component['remark']}}
                                   </td>
                                   <td class="text-muted w-5">
                                       <a target="_blank" href="/admin/hook/components/preview?component={{$component['import']}}">预览</a>
                                   </td>
                                   <td class="text-muted w-5">
                                       <a href="/admin/hook/components/edit?path={{$component['file_name']}}">编辑</a>
                                   </td>
                               </tr>
                           @endforeach
                       @endif
                       </tbody>
                   </table>
               </div>
               <div class="mt-3">
                   {!! make_page($page) !!}
               </div>
           </div>
       </div>
       @if(!$page->count())
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