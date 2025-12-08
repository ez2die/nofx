#!/usr/bin/env python3
"""
分析决策日志中的止损问题
检查是否有频繁的过早止损
"""

import json
import os
from pathlib import Path
from datetime import datetime
from collections import defaultdict
import glob
import re

def analyze_stop_loss_logs(log_dir, num_logs=400):
    """分析最新的N个日志文件中的止损情况"""
    
    # 获取所有日志文件并按时间排序
    log_files = sorted(glob.glob(os.path.join(log_dir, "decision_*.json")), reverse=True)
    log_files = log_files[:num_logs]
    
    print(f"分析最新的 {len(log_files)} 个日志文件...\n")
    
    # 统计数据
    stop_loss_records = []
    all_trades = []
    position_snapshots = {}  # symbol -> {entry_price, stop_loss, take_profit, entry_time}
    
    for log_file in log_files:
        try:
            with open(log_file, 'r', encoding='utf-8') as f:
                data = json.load(f)
            
            cycle_num = data.get('cycle_number', 0)
            timestamp = data.get('timestamp', '')
            
            # 提取持仓快照（入场信息）
            positions = data.get('positions', [])
            if positions is None:
                positions = []
            if positions:
                for pos in positions:
                    symbol = pos.get('symbol', '')
                    entry_price = pos.get('entry_price', 0)
                    mark_price = pos.get('mark_price', 0)
                    side = pos.get('side', '')
                    
                    # 从input_prompt中提取止损止盈信息
                    input_prompt = data.get('input_prompt', '')
                    stop_loss = None
                    take_profit = None
                    
                    if '入场快照' in input_prompt:
                        # 解析入场快照
                        lines = input_prompt.split('\n')
                        for i, line in enumerate(lines):
                            if '入场快照' in line and i + 1 < len(lines):
                                snapshot_line = lines[i + 1] if i + 1 < len(lines) else ''
                                if '止损' in snapshot_line and '止盈' in snapshot_line:
                                    # 提取止损和止盈价格
                                    stop_match = re.search(r'止损\s+([\d.]+)', snapshot_line)
                                    tp_match = re.search(r'止盈\s+([\d.]+)', snapshot_line)
                                    if stop_match:
                                        stop_loss = float(stop_match.group(1))
                                    if tp_match:
                                        take_profit = float(tp_match.group(1))
                    
                    if symbol and entry_price:
                        key = f"{symbol}_{side}"
                        position_snapshots[key] = {
                            'entry_price': entry_price,
                            'stop_loss': stop_loss,
                            'take_profit': take_profit,
                            'entry_time': timestamp,
                            'cycle': cycle_num
                        }
            
            # 提取决策记录
            decisions = data.get('decisions', [])
            if decisions is None:
                decisions = []
            for decision in decisions:
                action = decision.get('action', '')
                symbol = decision.get('symbol', '')
                price = decision.get('price', 0)
                timestamp_dec = decision.get('timestamp', '')
                is_auto = decision.get('is_auto_triggered', False)
                was_stop_loss = decision.get('was_stop_loss', False)
                
                if action in ['open_long', 'open_short']:
                    all_trades.append({
                        'cycle': cycle_num,
                        'symbol': symbol,
                        'action': action,
                        'price': price,
                        'timestamp': timestamp_dec
                    })
                
                if is_auto and was_stop_loss:
                    # 这是止损记录
                    key = f"{symbol}_{'long' if 'close_long' in action else 'short'}"
                    entry_info = position_snapshots.get(key, {})
                    
                    entry_price = entry_info.get('entry_price', 0)
                    stop_loss_price = entry_info.get('stop_loss', 0)
                    
                    # 计算止损距离
                    if entry_price > 0:
                        if 'long' in action:
                            stop_distance_pct = ((entry_price - price) / entry_price) * 100
                        else:
                            stop_distance_pct = ((price - entry_price) / entry_price) * 100
                    else:
                        stop_distance_pct = 0
                    
                    stop_loss_records.append({
                        'cycle': cycle_num,
                        'symbol': symbol,
                        'action': action,
                        'entry_price': entry_price,
                        'stop_loss_set': stop_loss_price,
                        'actual_close_price': price,
                        'stop_distance_pct': stop_distance_pct,
                        'timestamp': timestamp_dec,
                        'entry_time': entry_info.get('entry_time', '')
                    })
        
        except Exception as e:
            print(f"处理文件 {log_file} 时出错: {e}")
            continue
    
    # 分析结果
    print("=" * 80)
    print("止损分析报告")
    print("=" * 80)
    print(f"\n总止损次数: {len(stop_loss_records)}")
    print(f"总交易次数: {len(all_trades)}")
    
    if len(stop_loss_records) == 0:
        print("\n✅ 未发现止损记录")
        return
    
    # 按币种统计
    symbol_stats = defaultdict(lambda: {'count': 0, 'distances': []})
    for record in stop_loss_records:
        symbol = record['symbol']
        symbol_stats[symbol]['count'] += 1
        symbol_stats[symbol]['distances'].append(record['stop_distance_pct'])
    
    print("\n按币种统计止损次数:")
    for symbol, stats in sorted(symbol_stats.items(), key=lambda x: x[1]['count'], reverse=True):
        avg_distance = sum(stats['distances']) / len(stats['distances']) if stats['distances'] else 0
        min_distance = min(stats['distances']) if stats['distances'] else 0
        max_distance = max(stats['distances']) if stats['distances'] else 0
        print(f"  {symbol}: {stats['count']}次, 平均止损距离: {avg_distance:.3f}%, 最小: {min_distance:.3f}%, 最大: {max_distance:.3f}%")
    
    # 检查过早止损模式
    print("\n" + "=" * 80)
    print("过早止损分析")
    print("=" * 80)
    
    premature_stops = []
    for record in stop_loss_records:
        symbol = record['symbol']
        distance = record['stop_distance_pct']
        
        # 判断标准：
        # 1. BTC/ETH: 止损距离 < 0.3% 可能是过早止损
        # 2. 大市值山寨币: 止损距离 < 0.5%
        # 3. 其他: 止损距离 < 0.7%
        
        min_distance = 0.3 if symbol in ['BTCUSDT', 'ETHUSDT'] else 0.5 if symbol in ['SOLUSDT', 'BNBUSDT'] else 0.7
        
        if distance < min_distance:
            premature_stops.append(record)
    
    print(f"\n可能的过早止损次数: {len(premature_stops)} / {len(stop_loss_records)} ({len(premature_stops)/len(stop_loss_records)*100:.1f}%)")
    
    if premature_stops:
        print("\n过早止损详情:")
        for record in premature_stops[:20]:  # 显示前20个
            print(f"  Cycle {record['cycle']}: {record['symbol']} {record['action']}")
            stop_loss_str = f"{record['stop_loss_set']:.2f}" if record['stop_loss_set'] else 'N/A'
            print(f"    入场价: {record['entry_price']:.2f}, 止损价: {stop_loss_str}")
            print(f"    实际平仓价: {record['actual_close_price']:.2f}, 止损距离: {record['stop_distance_pct']:.3f}%")
            print(f"    时间: {record['timestamp']}")
            print()
    
    # 检查止损后价格反弹的情况（需要对比后续日志）
    print("\n" + "=" * 80)
    print("止损后价格反弹分析（需要更多数据）")
    print("=" * 80)
    print("注意: 此分析需要对比止损后的价格走势，当前仅统计止损记录")
    
    # 总结
    print("\n" + "=" * 80)
    print("总结")
    print("=" * 80)
    if len(premature_stops) > len(stop_loss_records) * 0.3:
        print(f"⚠️  警告: {len(premature_stops)/len(stop_loss_records)*100:.1f}% 的止损可能是过早止损")
        print("建议:")
        print("  1. 检查止损设置是否考虑了市场噪音")
        print("  2. BTC/ETH最小止损距离应 ≥ 0.3%")
        print("  3. 大市值山寨币最小止损距离应 ≥ 0.5%")
        print("  4. 其他币种最小止损距离应 ≥ 0.7%")
    else:
        print(f"✅ 过早止损比例较低 ({len(premature_stops)/len(stop_loss_records)*100:.1f}%)")
    
    return stop_loss_records, premature_stops

if __name__ == "__main__":
    log_dir = "/root/nofx/decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728"
    analyze_stop_loss_logs(log_dir, num_logs=400)

