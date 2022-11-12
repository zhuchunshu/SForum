<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreatePayOrderTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('pay_order', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->text('title')->comment('订单标题');
            $table->string('status')->comment('订单状态');
            $table->string('user_id')->comment('订单发起者');
            $table->string('amount')->comment('订单金额');
            //$table->string('')
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('pay_order');
    }
}
