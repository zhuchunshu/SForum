@extends('app')

@section('title','部件管理')

@section('content')
   <div class="row row-cards">
       <div class="col-lg-12">
           <div class="card">
               <div class="card-header">
                   <h3 class="card-title">部件列表</h3>
                   <div class="card-actions">
                       <a href="/admin/hook/components/create">新建</a>
                   </div>
               </div>
               <div class="table-responsive" id="vue-admin-hook-components">
                   <table class="table card-table table-vcenter">
                       <thead>
                       <tr>
                           <th>#</th>
                           <th>调用代码</th>
                           <th>备注</th>
                           <th></th>
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
                                   <td class="text-muted w-5">
                                       <a @@click="rm('{{$component['file_name']}}')" href="#">删除</a>
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
   </div>
@endsection

@section('scripts')
    <script src="{{mix('js/admin/component.js')}}"></script>
@endsection