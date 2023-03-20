@extends('app')

@section('title','部件管理')

@section('content')
   <div class="row row-cards">
       <div class="col-12">
           <div class="alert alert-info alert-dismissible" role="alert">
               <div class="d-flex">
                   <div>
                       <!-- Download SVG icon from http://tabler-icons.io/i/info-circle -->
                       <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><circle cx="12" cy="12" r="9" /><line x1="12" y1="8" x2="12.01" y2="8" /><polyline points="11 12 12 12 12 16 13 16" /></svg>
                   </div>
                   <div>
                       <div class="text-muted">SForum已弃用热重载功能，修改后记得重启SForum服务，不然不会生效，严重则产生错误。 </div>
                   </div>
               </div>
               <a class="btn-close" data-bs-dismiss="alert" aria-label="close"></a>
           </div>
       </div>
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