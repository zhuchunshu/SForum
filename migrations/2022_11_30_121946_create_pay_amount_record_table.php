<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreatePayAmountRecordTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('pay_amount_record', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->string('original')->comment('原余额');
            $table->string('cash')->comment('现余额');
            $table->string('user_id');
            $table->string('order_id')->comment('绑定订单号')->nullable();
            $table->text('remark')->comment('备注')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('pay_amount_record_table');
    }
}
